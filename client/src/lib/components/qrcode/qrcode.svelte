<script lang="ts">
    import QRCode from 'qrcode';
    import { onMount } from 'svelte';

    let { value = '', size = 128 } = $props();
    let dataUrl = $state('');

    onMount(async () => {
        updateQRCode();
    });

    async function updateQRCode() {
        if (value) {
            QRCode.toDataURL(value, {
                margin: 0,
                errorCorrectionLevel: 'L',
            }).then((url: string) => (dataUrl = url));
        }
    }

    $effect(() => {
        updateQRCode();
    });
</script>

{#if dataUrl}
    <img alt="QR code" src={dataUrl} width={size} height={size} />
{/if}
