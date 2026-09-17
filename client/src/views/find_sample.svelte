<script lang="ts">
    import { Info } from 'lucide-svelte';
    import { toast } from 'svelte-sonner';

    import { AppStore } from '$lib/components/app_store';
    import QRScanner from '$lib/components/qr_scanner/qr_scanner.svelte';
    import * as Alert from '$lib/components/ui/alert';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';

    let { active = $bindable(false) } = $props();

    let selectedVideoInput: string = $state('');
    // No need to enumerate video inputs here; handled by QRScanner

    function handleManualSampleId(sampleId: string) {
        window.location.assign(`/app?sample_id=${sampleId}`);
    }

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
    <div class="flex-row flex items-center justify-center gap-4 flex-wrap">
        <div class="relative block min-w-[50%] h-full">
            <QRScanner
                containerId="qr-reader-find"
                bind:selectedVideoInput={selectedVideoInput}
                onQrCodeScan={handleQRScan}
                onManualSampleId={handleManualSampleId}
                manualIdInputId="id-input"
                autoStart={active}>
            </QRScanner></div>

        <div class="flex flex-col items-center grow">
            <Alert.Root class="max-w-xl">
                <Info />
                <Alert.Title
                    >Want to apply changes to many samples at once?</Alert.Title>
                <Alert.Description>
                    Try the Bulk Apply feature to apply changes to multiple
                    samples simultaneously.
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
</div>

<style></style>
