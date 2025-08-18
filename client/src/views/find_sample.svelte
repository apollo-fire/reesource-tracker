<script lang="ts">
    import { Info } from 'lucide-svelte';
    import { toast } from 'svelte-sonner';

    import { AppStore } from '$lib/components/app_store';
    import QRScanner from '$lib/components/qr_scanner/qr_scanner.svelte';
    import * as Alert from '$lib/components/ui/alert';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import * as InputOTP from '$lib/components/ui/input-otp';
    import { Label } from '$lib/components/ui/label';

    let { active = $bindable(false) } = $props();

    let selectedVideoInput: string = $state('');
    let new_sample: string = $state('');

    $effect(() => {
        if (new_sample.length == 6) {
            window.location.assign(
                `/app?sample_id=${new_sample.slice(0, 2)}-${new_sample.slice(2, 4)}-${new_sample.slice(4, 6)}`,
            );
        }
    });
    // No need to enumerate video inputs here; handled by QRScanner

    // QR scan handler
    function handleQRScan(text: string) {
        if (!active) return;
        try {
            const url = new URL(text);
            // Only open if same host
            if (url.host === window.location.host) {
                window.location.href = url.href;
            } else {
                toast.error('Scanned QR code is not for this host.');
            }
        } catch (_) {
            toast.error('Scanned QR code is not a valid URL.');
        }
    }

    // No need to enumerate video inputs here; handled by QRScanner
</script>

<div class="space-y-4 flex flex-col min-h-full justify-stretch">
    <Card.Root class="flex-grow">
        <Card.Header>
            <Card.Title>Find Sample</Card.Title>
            <Card.Description>
                Scan a QR code or manually enter the sample ID to go to the
                sample's details page.
            </Card.Description>
        </Card.Header>
        <Card.Content>
            <QRScanner
                containerId="qr-reader-find"
                bind:selectedVideoInput={selectedVideoInput}
                onQrCodeScan={handleQRScan}
                autoStart={active} />
            <div class="self-center mt-12 flex flex-col items-center gap-6">
                <Label for="id-input">Or manually enter the sample ID</Label>
                <InputOTP.Root
                    maxlength={6}
                    bind:value={new_sample}
                    id="id-input">
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

                <div class="flex flex-col items-center mt-12">
                    <Alert.Root>
                        <Info />
                        <Alert.Title
                            >Want to apply changes to many samples at once?</Alert.Title>
                        <Alert.Description>
                            Try the Bulk Apply feature to apply changes to
                            multiple samples simultaneously.
                            <div class="flex flex-row justify-end w-full">
                                <Button
                                    variant="outline"
                                    class="mt-2"
                                    onclick={() => {
                                        $AppStore.currentPage = 'bulk_apply';
                                        $AppStore = $AppStore; // Trigger reactivity
                                    }}>Try Bulk Apply</Button>
                            </div>
                        </Alert.Description>
                    </Alert.Root>
                </div>
            </div>
        </Card.Content>
    </Card.Root>
</div>

<style>
    #qr-reader-find {
        width: 100%;
        max-width: 400px;
        margin: auto;
    }
</style>
