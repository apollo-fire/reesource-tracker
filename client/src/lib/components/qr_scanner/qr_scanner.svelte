<script lang="ts">
    import QrScanner from 'qr-scanner';
    import { onDestroy, onMount, tick } from 'svelte';

    import { Label } from '$lib/components/ui/label';
    import * as Select from '$lib/components/ui/select/index.js';
    import { Switch } from '$lib/components/ui/switch/index.js';

    let videoInputs: MediaDeviceInfo[] = $state([]);
    let {
        selectedVideoInput = $bindable(''),
        containerId = 'qr-reader',
        autoStart = $bindable(false),
        flipFrontCamera = $bindable(true),
        onQrCodeScan = $bindable((text: string) => {}),
    } = $props();
    let allowCamera = $state(
        localStorage.getItem('qrScannerCameraOn') === 'true',
    );

    let videoElement: HTMLVideoElement | null = $state(null);
    let flipVideo = $state(false);
    let showVideo = $state(false);
    let videoInputError: string = $state('');

    let qrScanner: QrScanner | null = null;

    $inspect(videoElement);

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

    onDestroy(() => {
        stopScanner();
    });
</script>

<div class="flex flex-col items-center justify-center">
    <div id={containerId} class="h-[20vh]">
        {#if showVideo}
            <video
                autoplay
                playsinline
                class="max-h-full max-w-full rounded-lg shadow"
                bind:this={videoElement}
                style={flipVideo ? 'transform: scaleX(-1);' : ''}>
                <track
                    kind="captions"
                    src=""
                    srcLang="en"
                    label="English"
                    default />
            </video>
        {/if}
    </div>
    <p class="text-sm text-gray-500 mt-6">
        QR Codes are scanned and processed locally on your device. No data is
        sent to the server.
    </p>
    <div class="mt-2 flex items-center gap-2">
        <Label class="block font-bold" for="camera-select">Camera</Label>
        {#if videoInputError}
            <div class="text-destructive">{videoInputError}</div>
        {:else}
            <Select.Root type="single" bind:value={selectedVideoInput}>
                <Select.Trigger class="w-full"
                    >{videoInputs.find((d) => d.deviceId === selectedVideoInput)
                        ?.label ||
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
        <div class="ml-4 flex items-center gap-2">
            <Switch bind:checked={allowCamera} id="camera-switch"></Switch>
            <Label for="camera-switch" class="select-none cursor-pointer">
                {allowCamera ? 'Camera On' : 'Camera Off'}
            </Label>
        </div>
    </div>
</div>

<style>
    .qr-reader {
        width: 100%;
        max-width: 400px;
        margin: auto;
    }
</style>
