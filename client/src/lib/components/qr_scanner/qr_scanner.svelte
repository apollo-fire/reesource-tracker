<script lang="ts">
    import { Info } from 'lucide-svelte';
    import QrScanner from 'qr-scanner';
    import { onDestroy, onMount, tick } from 'svelte';
    import type { Snippet } from 'svelte';

    import { Card } from '$lib/components/ui/card/index.js';
    import * as InputOTP from '$lib/components/ui/input-otp';
    import { Label } from '$lib/components/ui/label';
    import * as Select from '$lib/components/ui/select/index.js';
    import { Switch } from '$lib/components/ui/switch/index.js';
    import * as Tooltip from '$lib/components/ui/tooltip';

    let videoInputs: MediaDeviceInfo[] = $state([]);
    let {
        selectedVideoInput = $bindable(''),
        containerId = 'qr-reader',
        autoStart = $bindable(false),
        flipFrontCamera = $bindable(true),
        onQrCodeScan = $bindable((_: string) => {}),
        onManualSampleId = $bindable((_: string) => {}),
        manualIdInputId = 'manual-id-input',
        manualIdInputDisabled = false,
        manualSampleError = '',
        children,
    }: {
        selectedVideoInput?: string;
        containerId?: string;
        autoStart?: boolean;
        flipFrontCamera?: boolean;
        onQrCodeScan?: (value: string) => void;
        onManualSampleId?: (sampleId: string) => void;
        manualIdInputId?: string;
        manualIdInputDisabled?: boolean;
        manualSampleError?: string;
        children?: Snippet;
    } = $props();
    let allowCamera = $state(
        localStorage.getItem('qrScannerCameraOn') === 'true',
    );
    let manualSample: string = $state('');

    let videoElement: HTMLVideoElement | null = $state(null);
    let flipVideo = $state(false);
    let showVideo = $state(false);
    let videoInputError: string = $state('');

    let qrScanner: QrScanner | null = null;

    async function startScanner() {
        if (videoElement) return;
        showVideo = true;
        await tick(); // wait for svelte to render the current application state
        if (!videoElement) return;
        // Flip preview if front-facing camera is selected
        if (flipFrontCamera && videoInputs && selectedVideoInput) {
            const selectedDevice = videoInputs.find(
                (d) => d.deviceId === selectedVideoInput,
            );
            if (
                selectedDevice &&
                /front|user|integrated/i.test(selectedDevice.label)
            ) {
                flipVideo = true;
            } else {
                flipVideo = false;
            }
        }
        // Stop any previous scanner
        if (qrScanner) {
            qrScanner.destroy();
            qrScanner = null;
        }
        qrScanner = new QrScanner(
            videoElement,
            (result) => {
                if (result) {
                    onQrCodeScan(result.data);
                }
            },
            {
                preferredCamera: selectedVideoInput || undefined,
                highlightScanRegion: false,
                highlightCodeOutline: true,
                maxScansPerSecond: 30,
                calculateScanRegion(video) {
                    return {
                        x: 0,
                        y: 0,
                        width: video.videoWidth,
                        height: video.videoHeight,
                    };
                },
                // Use high-res constraints for the stream
            },
        );
        await qrScanner.start();
    }

    async function stopScanner() {
        if (qrScanner) {
            qrScanner.destroy();
            qrScanner = null;
        }
        showVideo = false;
        await tick();
    }

    async function restartScanner() {
        if (localStorage.getItem('qrScannerCameraOn') === 'true') {
            allowCamera = true;
            await stopScanner();
            await startScanner();
        } else {
            allowCamera = false;
        }
    }

    async function enumerateVideoInputs() {
        try {
            const devices = await navigator.mediaDevices.enumerateDevices();
            videoInputs = devices.filter((d) => d.kind === 'videoinput');
            if (videoInputs.length > 0) {
                if (
                    !selectedVideoInput ||
                    !videoInputs.find((d) => d.deviceId === selectedVideoInput)
                ) {
                    selectedVideoInput = videoInputs[0].deviceId;
                }
                videoInputError = '';
            } else {
                videoInputError = 'No video input devices found.';
                selectedVideoInput = '';
            }
        } catch (_) {
            videoInputError = 'Unable to enumerate video devices.';
            videoInputs = [];
            selectedVideoInput = '';
        }
    }

    onMount(async () => {
        await enumerateVideoInputs();
    });

    $effect(() => {
        if (allowCamera) {
            // Persist camera state in localStorage
            localStorage.setItem('qrScannerCameraOn', 'true');
            if (!qrScanner && autoStart) startScanner();
        } else {
            // Persist camera state in localStorage
            localStorage.setItem('qrScannerCameraOn', 'false');
            stopScanner();
        }
    });

    $effect(() => {
        if (autoStart && selectedVideoInput) {
            restartScanner();
        } else if (!autoStart) {
            stopScanner();
        }
    });

    $effect(() => {
        if (manualSample.length === 6) {
            const sampleId =
                `${manualSample.slice(0, 2)}-${manualSample.slice(2, 4)}-${manualSample.slice(4, 6)}`.toUpperCase();
            onManualSampleId(sampleId);
            manualSample = '';
        }
    });

    onDestroy(() => {
        stopScanner();
    });
</script>

<Card class="aspect-square max-h-[80vh] relative p-0 pb-4 overflow-visible">
    <div class="absolute top-2 left-2 z-10 overflow-visible">
        <Tooltip.Root>
            <Tooltip.Trigger
                class="inline-flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-3 outline-none"
                aria-label="QR scan privacy information">
                <Info class="size-4" />
            </Tooltip.Trigger>
            <Tooltip.Content side="bottom">
                <p>
                    QR Codes are scanned and processed locally on your device.
                    No data is sent to the server.
                </p>
            </Tooltip.Content>
        </Tooltip.Root>
    </div>
    <div
        class="relative h-[65vh] max-h-[calc(100vh-12rem)] overflow-hidden bg-gray-50 flex-1">
        <div
            class="h-full w-full absolute top-0 left-0 flex flex-col items-center justify-center">
            {#if showVideo}
                <video
                    autoplay
                    playsinline
                    class="w-full h-full object-cover"
                    bind:this={videoElement}
                    style={flipVideo ? 'transform: scaleX(-1);' : ''}>
                    <track
                        kind="captions"
                        src=""
                        srcLang="en"
                        label="English"
                        default />
                </video>
            {:else}
                <p class="text-gray-500">Camera is off</p>
            {/if}
        </div>

        <div
            class="absolute inset-x-0 bottom-0 z-10 flex flex-row flex-wrap justify-center gap-2 p-2 pointer-events-none items-center">
            {#if videoInputError}
                <div class="text-destructive">{videoInputError}</div>
            {:else}
                <Select.Root type="single" bind:value={selectedVideoInput}>
                    <Select.Trigger
                        class="bg-background/90 backdrop-blur-2xl z-10 pointer-events-auto max-h-none h-full mb-0"
                        >{videoInputs.find(
                            (d) => d.deviceId === selectedVideoInput,
                        )?.label ||
                            `Camera ${selectedVideoInput}`}</Select.Trigger>
                    <Select.Content>
                        {#each videoInputs as device (device.deviceId)}
                            <Select.Item value={device.deviceId}>
                                {device.label || `Camera ${device.deviceId}`}
                            </Select.Item>
                        {/each}
                    </Select.Content>
                </Select.Root>
            {/if}
            <Card
                class="pointer-events-auto bg-background/90 backdrop-blur-2xl p-2.5 gap-2">
                <div class="flex items-center gap-2 justify-center">
                    <Switch bind:checked={allowCamera} id="camera-switch"
                    ></Switch>
                    <Label
                        for="camera-switch"
                        class="select-none cursor-pointer text-xs">
                        {allowCamera ? 'Camera On' : 'Camera Off'}
                    </Label>
                </div>
            </Card>
            {#if children}
                <Card
                    class="pointer-events-auto bg-background/70 backdrop-blur-2xl p-2.5 gap-2 rounded-lg">
                    {@render children()}
                </Card>
            {/if}
        </div>
    </div>
    <div class="self-center flex flex-col items-center gap-2">
        <Label class="text-xs" for={manualIdInputId}
            >Or manually enter the sample ID</Label>
        <InputOTP.Root
            maxlength={6}
            bind:value={manualSample}
            id={manualIdInputId}
            disabled={manualIdInputDisabled}>
            {#snippet children({ cells })}
                <InputOTP.Group>
                    {#each cells.slice(0, 2) as cell (cell)}
                        <InputOTP.Slot cell={cell} />
                    {/each}
                </InputOTP.Group>
                <InputOTP.Separator />
                <InputOTP.Group>
                    {#each cells.slice(2, 4) as cell (cell)}
                        <InputOTP.Slot cell={cell} />
                    {/each}
                </InputOTP.Group>
                <InputOTP.Separator />
                <InputOTP.Group>
                    {#each cells.slice(4, 6) as cell (cell)}
                        <InputOTP.Slot cell={cell} />
                    {/each}
                </InputOTP.Group>
            {/snippet}
        </InputOTP.Root>
        {#if manualSampleError}
            <span class="text-red-500 text-sm">{manualSampleError}</span>
        {/if}
    </div></Card>

<style></style>
