<script lang="ts">
    import { onDestroy, onMount } from 'svelte';

    import { AppStore } from '$lib/components/app_store';
    import LocationSelect from '$lib/components/selects/location-select.svelte';
    import ProductSelect from '$lib/components/selects/product-select.svelte';
    import StateSelect from '$lib/components/selects/state-select.svelte';
    import UserSelect from '$lib/components/selects/user-select.svelte';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Checkbox } from '$lib/components/ui/checkbox';
    import { Input } from '$lib/components/ui/input';
    import { Label } from '$lib/components/ui/label';
    import * as Pagination from '$lib/components/ui/pagination';
    import * as Select from '$lib/components/ui/select';
    import { Separator } from '$lib/components/ui/separator';
    import * as Sheet from '$lib/components/ui/sheet/index.js';
    import * as Table from '$lib/components/ui/table';

    let selectedProduct = $state('');
    let selectedLocation = $state('');
    let page = $state(1);

    let pageSize = $state(1);
    let tableContainer: HTMLDivElement | null = $state(null);
    let firstRow: HTMLElement | null = $state(null);
    let paginationContainer: HTMLDivElement | null = $state(null);

    let showUnassignedOrArchived = $state(false);
    let modQuery = $state('');
    let moddedState = $state('any'); // New filter for modded state
    let selectedState = $state('');
    let ownerQuery = $state(''); // Will hold user id
    let issueQuery = $state('');

    function calculatePageSize() {
        if (!tableContainer) {
            console.warn(
                'tableContainer not available, cannot calculate page size',
            );
            setTimeout(() => {
                // Fallback in case firstRow is not immediately available
                calculatePageSize();
            }, 1000);
            return;
        }
        tableContainer.style.height = '100%';

        if (!firstRow) {
            // Fallback if firstRow is not available
            pageSize = 1;
            console.warn(
                'firstRow not available, using fallback pageSize of 1',
            );
            setTimeout(() => {
                // Fallback in case firstRow is not immediately available
                calculatePageSize();
            }, 1000);
            return;
        }

        let ROW_HEIGHT = 9999;
        const measured = firstRow.clientHeight;
        if (measured && measured > 0) {
            ROW_HEIGHT = measured;
        }
        const totalHeight =
            tableContainer.clientHeight -
                (paginationContainer?.clientHeight || 0) || 0;
        const rows = Math.max(1, Math.floor(totalHeight / ROW_HEIGHT));
        pageSize = rows;
    }

    let resizeObserver: ResizeObserver | null = null;

    onMount(() => {
        calculatePageSize();
        if (tableContainer) {
            resizeObserver = new ResizeObserver(() => {
                calculatePageSize();
            });
            resizeObserver.observe(tableContainer);
        }
        window.addEventListener('resize', calculatePageSize);
    });

    onDestroy(() => {
        if (resizeObserver && tableContainer) {
            resizeObserver.unobserve(tableContainer);
            resizeObserver.disconnect();
        }
        window.removeEventListener('resize', calculatePageSize);
    });

    // Helper to get all descendant ids for a given parent id (recursive)
    function getDescendantProductIds(parentId: string): string[] {
        const directChildren = $AppStore.products.filter(
            (p) => p.parentProductID === parentId,
        );
        let allIds: string[] = directChildren.map((p) => p.id);
        for (const child of directChildren) {
            allIds = allIds.concat(getDescendantProductIds(child.id));
        }
        return allIds;
    }
    function getDescendantLocationIds(parentId: string): string[] {
        const directChildren = $AppStore.locations.filter(
            (l) => l.parentLocationID === parentId,
        );
        let allIds: string[] = directChildren.map((l) => l.id);
        for (const child of directChildren) {
            allIds = allIds.concat(getDescendantLocationIds(child.id));
        }
        return allIds;
    }

    const filteredSamples = $derived(() => {
        let result = $AppStore.samples;
        if (
            !showUnassignedOrArchived &&
            selectedState !== 'unassigned' &&
            selectedState !== 'archived'
        ) {
            result = result.filter((sample) => {
                return (
                    sample.state !== 'unassigned' && sample.state !== 'archived'
                );
            });
        }
        if (selectedState && selectedState !== '') {
            result = result.filter((sample) => sample.state === selectedState);
        }
        if (selectedProduct) {
            if (selectedProduct.startsWith('any-')) {
                const parentId = selectedProduct.slice(4);
                const descendantIds = getDescendantProductIds(parentId);
                // Include both parent and all descendants
                const validIds = [parentId, ...descendantIds];
                result = result.filter(
                    (sample) =>
                        sample.Product?.id &&
                        validIds.includes(sample.Product.id),
                );
            } else {
                result = result.filter(
                    (sample) => sample.Product?.id === selectedProduct,
                );
            }
        }

        if (selectedLocation) {
            if (selectedLocation.startsWith('any-')) {
                const parentId = selectedLocation.slice(4);
                const descendantIds = getDescendantLocationIds(parentId);
                // Include both parent and all descendants
                const validIds = [parentId, ...descendantIds];
                result = result.filter(
                    (sample) =>
                        sample.Location?.id &&
                        validIds.includes(sample.Location.id),
                );
            } else {
                result = result.filter(
                    (sample) => sample.Location?.id === selectedLocation,
                );
            }
        }
        // Modded state filter
        if (moddedState !== 'any') {
            result = result.filter((sample) => {
                const summary = sample.ModSummary?.toLowerCase() || '';
                if (moddedState === 'active') {
                    return (
                        summary !== 'no mods' && summary !== 'no active mods'
                    );
                } else if (moddedState === 'noactive') {
                    // No active mods, but previously modded
                    return summary === 'no active mods';
                } else if (moddedState === 'never') {
                    // Never modded
                    return summary === 'no mods';
                }
                return true;
            });
        }
        if (modQuery != '') {
            const queries = modQuery
                .toLowerCase()
                .split(',')
                .map((q) => q.trim());
            result = result.filter((sample) => {
                if (sample.ModSummary === 'No mods') return false;
                if (sample.ModSummary === 'No active mods') return false;
                return queries.some((query) =>
                    sample.ModSummary.toLowerCase().includes(query),
                );
            });
        }
        if (ownerQuery && ownerQuery !== '') {
            result = result.filter((sample) => sample.Owner?.id === ownerQuery);
        }
        if (issueQuery.trim() !== '') {
            const iq = issueQuery.toLowerCase();
            result = result.filter((sample) =>
                sample.productIssue?.toLowerCase().includes(iq),
            );
        }
        return result;
    });

    const totalPages = $derived(() =>
        Math.max(1, Math.ceil(filteredSamples().length / pageSize)),
    );
    const paginatedSamples = $derived(() =>
        filteredSamples().slice((page - 1) * pageSize, page * pageSize),
    );

    $effect(() => {
        const _ =
            selectedLocation ||
            selectedProduct ||
            modQuery ||
            selectedState ||
            showUnassignedOrArchived ||
            issueQuery ||
            ownerQuery ||
            moddedState; // Effect dependencies

        page = 1; // Reset to first page when filters change
    });

    let lockModFilter = $derived(
        moddedState === 'never' || moddedState === 'noactive',
    );

    $effect(() => {
        if (lockModFilter) {
            modQuery = ''; // Clear mod query if modded state is locked
        }
    });

    function EditSample(sampleId: string) {
        window.location.href = `/app?sample_id=${sampleId}`;
    }
</script>

<div class="max-h-full grow h-full overflow-auto">
    <div class="flex flex-col h-full max-h-full">
        <Card.Root class="grow">
            <Card.Header>
                <Card.Title>Sample List</Card.Title>
                <Card.Description>
                    Find and manage samples in the system.
                </Card.Description>
                <div class="flex flex-row w-full justify-between items-center">
                    <Sheet.Root>
                        <Sheet.Trigger>
                            <Button variant="outline">Filters</Button>
                        </Sheet.Trigger>
                        <Sheet.Content
                            side="left"
                            class="max-w-max w-auto pl-1 pr-6">
                            <Sheet.Header>
                                <Sheet.Title>Sample Filters</Sheet.Title>
                                <Sheet.Description>
                                    Filter the sample list by product, issue,
                                    mods, location, owner, state, and more.
                                </Sheet.Description>
                            </Sheet.Header>
                            <div class="pl-3 pr-6">
                                <div class="flex flex-col gap-6">
                                    <div>
                                        <Label class="mb-2" for="product-select"
                                            >Product</Label>
                                        <ProductSelect
                                            bind:bindValue={selectedProduct}
                                            filterMode={true} />
                                    </div>
                                    <div>
                                        <Label class="mb-2" for="issue-query"
                                            >Issue</Label>
                                        <Input
                                            type="text"
                                            placeholder="Filter by issue"
                                            class="input input-bordered w-full"
                                            bind:value={issueQuery} />
                                    </div>
                                    <div>
                                        <Label class="mb-2" for="modded-state"
                                            >Modded State</Label>
                                        <Select.Root
                                            bind:value={moddedState}
                                            type="single">
                                            <Select.Trigger
                                                id="modded-state"
                                                class="w-full mb-2">
                                                {(() => {
                                                    switch (moddedState) {
                                                        case 'active':
                                                            return 'Active Mods';
                                                        case 'noactive':
                                                            return 'No Active Mods, Previously Modded';
                                                        case 'never':
                                                            return 'Never Modded';
                                                        default:
                                                            return 'Any';
                                                    }
                                                })()}
                                            </Select.Trigger>
                                            <Select.Content class="w-full">
                                                <Select.Item value="any"
                                                    >Any</Select.Item>
                                                <Select.Item value="active"
                                                    >Active Mods</Select.Item>
                                                <Select.Item value="noactive"
                                                    >No Active Mods, Previously
                                                    Modded</Select.Item>
                                                <Select.Item value="never"
                                                    >Never Modded</Select.Item>
                                            </Select.Content>
                                        </Select.Root>
                                        {#if !lockModFilter}
                                            <Input
                                                type="text"
                                                placeholder="Filter by specific mods"
                                                class="input input-bordered w-full"
                                                bind:value={modQuery}
                                                disabled={lockModFilter} />
                                        {/if}
                                    </div>
                                    <div>
                                        <Label
                                            class="mb-2"
                                            for="location-select"
                                            >Location</Label>
                                        <LocationSelect
                                            bind:bindValue={selectedLocation}
                                            filterMode={true} />
                                    </div>
                                    <div>
                                        <Label class="mb-2" for="owner-query"
                                            >Owner</Label>
                                        <UserSelect
                                            bind:bindValue={ownerQuery}
                                            filterMode={true} />
                                    </div>
                                    <div>
                                        <Label class="mb-2" for="state-select"
                                            >State</Label>
                                        <StateSelect
                                            bind:bindValue={selectedState}
                                            filterMode={true}
                                            showUnassignedOrArchived={true} />
                                        {#if selectedState === ''}
                                            <div
                                                class="flex mt-4 items-center gap-2">
                                                <Checkbox
                                                    id="show-unassigned"
                                                    bind:checked={
                                                        showUnassignedOrArchived
                                                    } />
                                                <Label
                                                    for="show-unassigned"
                                                    class="no-wrap min-w-max">
                                                    Include Unassigned or
                                                    Archived Samples
                                                </Label>
                                            </div>
                                        {/if}
                                    </div>
                                </div>
                            </div>
                        </Sheet.Content>
                    </Sheet.Root>
                    <Button
                        variant="outline"
                        onclick={() => {
                            $AppStore.currentPage = 'find_sample';
                            $AppStore = $AppStore;
                        }}>Find Sample by ID or QR Code</Button>
                </div>
            </Card.Header>
            <Separator />
            <Card.Content class="h-full flex flex-col">
                <div
                    bind:this={tableContainer}
                    class="grow basis-0 max-h-full h-full overflow-hidden">
                    <Table.Root>
                        <Table.Header>
                            <Table.Row>
                                <Table.Head>ID</Table.Head>
                                <Table.Head>Product</Table.Head>
                                <Table.Head>Issue</Table.Head>
                                <Table.Head>Mods</Table.Head>
                                <Table.Head>Location</Table.Head>
                                <Table.Head>Owner</Table.Head>
                                <Table.Head>Status</Table.Head>
                            </Table.Row>
                        </Table.Header>
                        <Table.Body>
                            {#each paginatedSamples() as sample, i (sample.id)}
                                <Table.Row
                                    bind:ref={
                                        () => firstRow,
                                        (el) =>
                                            i === 0 ? (firstRow = el) : null
                                    }>
                                    <Table.Cell>{sample.DisplayId}</Table.Cell>
                                    <Table.Cell
                                        >{sample.Product?.CombinedName}
                                        {#if sample.Product?.partNumber}{`(${sample.Product?.partNumber})`}{/if}</Table.Cell>
                                    <Table.Cell
                                        >{sample.productIssue}</Table.Cell>
                                    <Table.Cell
                                        class="max-w-32 overflow-hidden text-ellipsis">
                                        {sample.ModSummary}
                                    </Table.Cell>
                                    <Table.Cell
                                        >{sample.Location?.name}</Table.Cell>
                                    <Table.Cell
                                        >{sample.Owner?.name}</Table.Cell>
                                    <Table.Cell>{sample.Status}</Table.Cell>
                                    <Table.Cell>
                                        <Button
                                            variant="link"
                                            class="py-0 my-0"
                                            onclick={() =>
                                                EditSample(sample.DisplayId)}>
                                            Edit
                                        </Button>
                                    </Table.Cell>
                                </Table.Row>
                            {/each}
                        </Table.Body>
                    </Table.Root>
                </div>
                <!-- Pagination Controls -->
                <div
                    class="flex justify-center mt-6"
                    bind:this={paginationContainer}>
                    <Pagination.Root
                        count={filteredSamples().length}
                        perPage={pageSize}
                        bind:page={page}>
                        {#snippet children({ pages, currentPage })}
                            <Pagination.Content>
                                <Pagination.Item>
                                    <Pagination.PrevButton
                                        disabled={page === 1} />
                                </Pagination.Item>
                                {#each pages as pageObj (pageObj.key)}
                                    {#if pageObj.type === 'ellipsis'}
                                        <Pagination.Item>
                                            <Pagination.Ellipsis />
                                        </Pagination.Item>
                                    {:else}
                                        <Pagination.Item>
                                            <Pagination.Link
                                                page={pageObj}
                                                isActive={currentPage ===
                                                    pageObj.value}>
                                                {pageObj.value}
                                            </Pagination.Link>
                                        </Pagination.Item>
                                    {/if}
                                {/each}
                                <Pagination.Item>
                                    <Pagination.NextButton
                                        disabled={page === totalPages()} />
                                </Pagination.Item>
                            </Pagination.Content>
                        {/snippet}
                    </Pagination.Root>
                </div>
            </Card.Content>
        </Card.Root>
    </div>
</div>
