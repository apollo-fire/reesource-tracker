<script lang="ts">
    import { CircleAlertIcon } from 'lucide-svelte';
    import { onMount } from 'svelte';
    import { toast } from 'svelte-sonner';

    import QrCode from '$lib/components/qrcode/qrcode.svelte';
    import * as Alert from '$lib/components/ui/alert';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Input } from '$lib/components/ui/input';
    import { Label } from '$lib/components/ui/label';

    let numSamples = $state(1);
    let generating = $state(false);
    let printouts: { timestamp: number; ids: string[] }[] = $state([]);
    let currentSheet: string[] = $state([]);
    let showPrintOverlay = $state(false);
    let fontSize = $state(3); // mm
    // Sizing parameters (modifiable by user)
    let qrPadding = $state(2); // mm, now used as grid gap
    let qrSize = $state(15); // mm (QR code SVG size)

    // Derived sizes
    // Use window size in mm if print view is open, otherwise default to A4 size
    function pxToMm(px: number) {
        // 1in = 25.4mm, window.devicePixelRatio for scaling
        return ((px / (window.devicePixelRatio || 1)) * 25.4) / 96;
    }

    // Print layout state
    let printSheetWidth = 198;
    let printSheetHeight = 111.5;
    let printGridCols = 13;
    let printGridRows = 7;
    let printPerSheet = printGridCols * printGridRows;

    // The grid slot size is determined by qrSize only; height for text is handled in CSS
    let gridSlotSize = $derived(qrSize); // mm
    let gridSlotPx = $derived(`${gridSlotSize}mm`);
    let qrSvgPx = $derived(`${qrSize}mm`);
    let qrPadPx = $derived(`${qrPadding}mm`); // for gap only

    // Dynamically calculate columns and rows based on user sizing (preview mode)
    let sheetWidth = $derived(showPrintOverlay ? printSheetWidth : 198);
    let sheetHeight = $derived(showPrintOverlay ? printSheetHeight : 111.5);
    let gridCols = $derived(
        showPrintOverlay
            ? printGridCols
            : Math.max(
                  1,
                  Math.floor((sheetWidth + qrPadding) / (qrSize + qrPadding)),
              ),
    );
    let gridRows = $derived(
        showPrintOverlay
            ? printGridRows
            : Math.max(
                  1,
                  Math.floor((sheetHeight + qrPadding) / (qrSize + qrPadding)),
              ),
    );
    let perSheet = $derived(
        showPrintOverlay ? printPerSheet : gridCols * gridRows,
    );

    // Print event listeners
    function recalculatePrintLayout() {
        // For print, let CSS handle grid wrapping dynamically
        const probe = document.createElement('div');
        probe.style.position = 'absolute';
        probe.style.top = '0';
        probe.style.left = '0';
        probe.style.width = '100vw';
        probe.style.height = '100vh';
        probe.style.visibility = 'hidden';
        probe.style.pointerEvents = 'none';
        document.body.appendChild(probe);

        const rect = probe.getBoundingClientRect();
        printSheetWidth = pxToMm(rect.width);
        printSheetHeight = pxToMm(rect.height);
        document.body.removeChild(probe);

        // Let CSS grid auto-fit columns/rows for print
        printGridCols = 0;
        printGridRows = 0;
        printPerSheet = currentSheet.length;
    }

    function resetPrintLayout() {
        printSheetWidth = 198;
        printSheetHeight = 111.5;
        printGridCols = 13;
        printGridRows = 7;
        printPerSheet = printGridCols * printGridRows;
    }

    onMount(() => {
        window.addEventListener('beforeprint', recalculatePrintLayout);
        window.addEventListener('afterprint', resetPrintLayout);
    });

    onMount(() => {
        const cached = localStorage.getItem('sample_printouts');
        if (cached) {
            printouts = JSON.parse(cached);
        }
    });

    function savePrintouts() {
        // Only keep the 10 most recent printouts
        if (printouts.length > 10) printouts = printouts.slice(0, 10);
        localStorage.setItem('sample_printouts', JSON.stringify(printouts));
    }

    function GetSampleUrl(id: string) {
        return `${location.protocol}//${location.host}/app?sample_id=${id}`;
    }

    async function generateSamples(e: SubmitEvent) {
        e.preventDefault(); // Prevent default form submission
        generating = true;
        toast.loading('Generating samples...');
        const res = await fetch(
            `/api/generate_samples?num_samples=${numSamples}`,
        );
        if (!res.ok) {
            throw new Error('Failed to generate samples');
        }
        const data = await res.json();
        if (!data.sample_ids) throw new Error('No sample IDs returned');
        currentSheet = data.sample_ids;
        printouts = [
            { timestamp: Date.now(), ids: [...currentSheet] },
            ...printouts,
        ];
        if (printouts.length > 10) printouts = printouts.slice(0, 10);
        savePrintouts();
        generating = false;
        toast.success('Samples generated successfully!');
    }

    function SelectSheet(ids: string[]) {
        toast.loading('Loading selected samples...');
        (async () => {
            currentSheet = [];
            for (const id of ids) {
                currentSheet.push(id);
            }
            currentSheet = currentSheet; // Trigger reactivity
            showPrintOverlay = false; // Hide print overlay if showing
            toast.success('Selected samples loaded!');
        })();
    }

    function deletePrintout(idx: number) {
        printouts = printouts.filter((_, i) => i !== idx);
        savePrintouts();
    }

    function printSheet(ids: string[]) {
        currentSheet = ids;
        showPrintOverlay = true;
        // Wait for overlay to render, then print
        setTimeout(() => {
            // Recalculate print layout before printing
            recalculatePrintLayout();
            window.print();
            // Hide overlay after print (with a delay for print dialog to close)
            setTimeout(() => {
                showPrintOverlay = false;
                resetPrintLayout();
            }, 500);
        }, 50);
    }

    function formatDate(ts: number) {
        return new Date(ts).toLocaleString();
    }

    let previewSheet = $derived(
        currentSheet.slice(0, perSheet).map((id) => id || ''),
    );
</script>

<div class="flex flex-col gap-6" id="sample-form">
    <Card.Root>
        <Card.Header>
            <Card.Title>Previous Code Sheets</Card.Title>
            <Card.Description>
                Select a previously generated code sheet to print or view.
            </Card.Description>
            <Alert.Root variant="destructive">
                <CircleAlertIcon class="size-4" />
                <Alert.Title
                    >Warning - Delete code sheets after printing</Alert.Title>
                <Alert.Description
                    >Be careful to not reprint any existing code sheets, as this
                    would result in samples sharing the same entries.</Alert.Description>
            </Alert.Root>
        </Card.Header>
        <Card.Content>
            <div class="flex flex-row max-w-full overflow-x-auto gap-8">
                {#each printouts as p, i (p.ids)}
                    <Card.Root class="w-[32em]">
                        <Card.Header>
                            <Card.Title>
                                {formatDate(p.timestamp)} -
                                <span>{p.ids.length} samples</span>
                            </Card.Title>
                        </Card.Header>
                        <Card.Content>
                            <details>
                                <summary>Show IDs</summary>
                                <div
                                    style="font-family:inconsolata; font-size:0.95em; word-break: break-all; max-height: 10em; overflow-y: auto;">
                                    {p.ids.join(', ')}
                                </div>
                            </details>
                            <div class="flex flex-row justify-end gap-2">
                                <Button
                                    onclick={() => SelectSheet(p.ids)}
                                    size="sm">Select</Button>
                                <Button
                                    variant="destructive"
                                    onclick={() => deletePrintout(i)}
                                    size="sm"
                                    aria-label="Delete history item"
                                    >Delete</Button>
                            </div>
                        </Card.Content>
                    </Card.Root>
                {/each}
            </div>
        </Card.Content>
    </Card.Root>

    <div class="flex flex-row gap-6 w-full">
        <Card.Root class="grow">
            <Card.Header>
                <Card.Title>Sample Code Generator</Card.Title>
                <Card.Description>
                    Reserve a new set of sample codes for printing.
                </Card.Description>
            </Card.Header>
            <Card.Content>
                <form onsubmit={generateSamples}>
                    <div class="flex flex-col gap-4">
                        <Label>
                            Number of samples to generate:
                            <Input
                                type="number"
                                min="1"
                                max="1000"
                                bind:value={numSamples}
                                disabled={generating} />
                        </Label>
                        <div>
                            <Button type="submit" disabled={generating}>
                                {generating ? 'Generating...' : 'Generate'}
                            </Button>
                        </div>
                    </div>
                </form>
            </Card.Content>
        </Card.Root>
        <Card.Root class="grow">
            <Card.Header>
                <Card.Title>QR Code Settings</Card.Title>
                <Card.Description>
                    Adjust the QR code and font sizes for the generated codes.
                </Card.Description>
            </Card.Header>
            <Card.Content>
                <div>
                    QR Padding (mm):
                    <Input
                        type="number"
                        min="0"
                        max="10"
                        bind:value={qrPadding} />
                </div>
                <div>
                    QR Size (mm):
                    <Input type="number" min="5" max="50" bind:value={qrSize} />
                </div>
                <div>
                    Font Size (mm):
                    <Input
                        type="number"
                        min="1"
                        max="4"
                        step="0.1"
                        bind:value={fontSize} />
                </div>
            </Card.Content>
        </Card.Root>
    </div>

    {#if currentSheet.length}
        <Card.Root>
            <Card.Header>
                <Card.Title>Preview Output</Card.Title>
            </Card.Header>
            <Card.Content class=" max-h-[20vh] overflow-hidden">
                <div
                    class="qr-sheet print-sheet-only"
                    style="--grid-slot-size: {gridSlotPx}; --qr-svg-size: {qrSvgPx}; --qr-gap: {qrPadPx}; --grid-cols: {gridCols}; --grid-rows: {gridRows};">
                    {#each previewSheet as id, i (id)}
                        <div class="qr-item">
                            {#if previewSheet[i]}
                                <QrCode value={GetSampleUrl(previewSheet[i])} />
                                <div
                                    class="qr-id"
                                    style="font-size: {fontSize}mm;">
                                    {previewSheet[i]}
                                </div>
                            {/if}
                        </div>
                    {/each}
                </div>
            </Card.Content>
            <Card.Footer>
                <Button
                    onclick={() => printSheet(currentSheet)}
                    style="margin-bottom:1em;">Print This Sheet</Button>
            </Card.Footer>
        </Card.Root>
    {/if}
</div>

{#if showPrintOverlay}
    <div class="print-overlay" aria-hidden={!showPrintOverlay}>
        <div
            class="qr-sheet print-sheet-only"
            style="--grid-slot-size: {gridSlotPx}; --qr-svg-size: {qrSvgPx}; --qr-gap: {qrPadPx};">
            {#each currentSheet as id (id)}
                <div class="qr-item">
                    <QrCode value={GetSampleUrl(id)} />
                    <div class="qr-id" style="font-size: {fontSize}mm;"
                        >{id}</div>
                </div>
            {/each}
        </div>
    </div>
{/if}

<style>
    /* Unified grid for both preview and print */
    .qr-sheet {
        display: grid;
        grid-template-columns: repeat(
            var(--grid-cols, 13),
            var(--grid-slot-size, 15mm)
        );
        grid-template-rows: repeat(
            var(--grid-rows, 7),
            calc(var(--grid-slot-size, 15mm) + 2em)
        );
        gap: var(--qr-gap, 2mm);
        margin: 2rem auto 0 auto;
        width: calc(
            var(--grid-slot-size, 15mm) * var(--grid-cols, 13) +
                var(--qr-gap, 2mm) * (var(--grid-cols, 13) - 1)
        );
        height: calc(
            (var(--grid-slot-size, 15mm) + 2em) * var(--grid-rows, 7) +
                var(--qr-gap, 2mm) * (var(--grid-rows, 7) - 1)
        );
        max-width: 100vw;
        box-sizing: border-box;
        justify-content: flex-start;
        align-items: flex-start;
        background: white;
        place-items: start;
    }
    .qr-item {
        width: 100%;
        height: 100%;
        margin: 0;
        padding: 0;
        break-inside: avoid;
        page-break-inside: avoid;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        box-sizing: border-box;
        user-select: none;
        pointer-events: none;
        position: relative;
        overflow: hidden;
        :global(:not(.qr-item)) {
            flex: 1 1 0;
            max-width: 100%;
            max-height: 100%;
            aspect-ratio: 1;
            flex-shrink: 1;
            display: block;
        }
        .qr-id {
            font-family: 'inconsolata', monospace;
            font-weight: 1000;
            width: 100%;
            overflow: hidden;
            text-align: center;
            box-sizing: border-box;
            padding: 0;
            color: black;
            background: none;
            break-inside: avoid;
            page-break-inside: avoid;
            flex-shrink: 0;
        }
    }

    /* svg sizing now handled above in .qr-item :global(svg) */
    @media print {
        body *:not(.print-overlay) {
            display: none !important;
            pointer-events: none !important;
        }
        #sample-form {
            display: none !important;
            height: 0px;
        }
        .print-overlay {
            display: block !important;
            position: absolute;
            z-index: 99999;
            top: 0;
            left: 0;
            width: 100vw;
            min-height: 100vh;
            background: white;
            user-select: none;
            pointer-events: auto;
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        .print-sheet-only.qr-sheet {
            margin: 6mm auto 6mm auto;
            background: white;
            display: grid;
            grid-template-columns: repeat(
                auto-fit,
                minmax(var(--grid-slot-size, 15mm), 1fr)
            );
            grid-auto-flow: row;
            grid-auto-rows: calc(var(--grid-slot-size, 15mm) + 2em);
            gap: var(--qr-gap, 2mm);
            width: 100vw;
            overflow: visible;
            max-width: 100vw;
            box-sizing: border-box;
            justify-content: flex-start;
            align-items: flex-start;
            place-items: start;
        }
        .qr-item {
            break-inside: avoid;
            page-break-inside: avoid;
        }
        .qr-id {
            break-inside: avoid;
            page-break-inside: avoid;
        }
    }
</style>
