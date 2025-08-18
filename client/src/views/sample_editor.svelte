<script lang="ts">
    import { onMount } from 'svelte';
    import { toast } from 'svelte-sonner';

    import { AppStore } from '$lib/components/app_store';
    import { IDStringToBlob } from '$lib/components/id_helper';
    import { Sample } from '$lib/components/sample';
    import LocationSelect from '$lib/components/selects/location-select.svelte';
    import ProductSelect from '$lib/components/selects/product-select.svelte';
    import StateSelect from '$lib/components/selects/state-select.svelte';
    import UserSelect from '$lib/components/selects/user-select.svelte';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Checkbox } from '$lib/components/ui/checkbox/index.js';
    import { Input } from '$lib/components/ui/input';
    import { Label } from '$lib/components/ui/label';
    import * as Table from '$lib/components/ui/table';

    let sample: Sample = $state(new Sample({}));
    let showFittedOnly = $state(false);
    let sample_id = '';
    let product_id = $state('');
    let location_id = $state('');
    let sample_state = $state('');
    let new_mod_name = $state('');
    let adding_mod = $state(false);
    let add_mod_error = $state('');
    let product_issue = $state('');
    let owner_id = $state('');

    AppStore.subscribe((state) => {
        const this_sample_new = state.samples.find((s) => s.id === sample.id);

        if (this_sample_new && this_sample_new !== sample) {
            fetchSample();
        }
    });

    async function fetchSample() {
        const params = new URLSearchParams(window.location.search);
        const new_sample_id = params.get('sample_id') || '';
        if (!new_sample_id) {
            return;
        }
        // Fetch sample data
        const res = await fetch(`/api/sample/${new_sample_id}`);
        if (res.ok) {
            const json_data = await res.json();
            sample = new Sample(
                { ...json_data.sample, mods: json_data.mods },
                AppStore,
            );
            sample_id = sample.DisplayId;
            product_id = sample.Product?.id || '';
            location_id = sample.Location?.id || '';
            sample_state = sample.state || '';
            owner_id = sample.Owner?.id || '';
            product_issue = sample.productIssue || '';
        } else {
            sample = new Sample(
                {
                    ID: IDStringToBlob(new_sample_id),
                    State: 'available',
                },
                AppStore,
            );
            sample_id = new_sample_id;
        }
    }

    async function addMod() {
        if (!new_mod_name.trim()) return;
        adding_mod = true;
        add_mod_error = '';
        try {
            const res = await fetch(`/api/sample/${sample_id}/mods`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: new_mod_name }),
            });
            if (!res.ok) {
                add_mod_error = await res.text();
            } else {
                new_mod_name = '';
                await fetchSample();
                toast.success('Mod added successfully.');
            }
        } catch (e) {
            add_mod_error = `Failed to add mod (${e})`;
        }
        adding_mod = false;
    }

    async function removeMod(mod_id: string) {
        if (!mod_id) return;
        await fetch(`/api/sample/${sample_id}/mods/${mod_id}`, {
            method: 'DELETE',
        });
        await fetchSample();
        toast.success('Mod removed successfully.');
    }

    async function saveSample(e: SubmitEvent) {
        e.preventDefault();
        const form = new FormData();
        form.append('location_id', location_id);
        form.append('product_id', product_id);
        form.append('state', sample_state);
        form.append('owner_id', owner_id);
        form.append('product_issue', product_issue);
        const res = await fetch(`/api/sample/${sample_id}`, {
            method: 'POST',
            body: form,
        });
        if (!res.ok) {
            toast.error('Failed to update sample.');
            throw new Error(await res.text());
        }
        // Optionally, show a success message or redirect
        await fetchSample();
        toast.success('Sample updated successfully.');
    }

    onMount(fetchSample);
</script>

<form
    onsubmit={saveSample}
    class="flex flex-col gap-4 p-4 min-h-full max-h-full overflow-auto">
    <div
        class="flex flex-row gap-6 w-full flex-wrap h-full overflow-auto flex-grow">
        <Card.Root class="flex-grow">
            <Card.Header>
                <Card.Title>Edit Sample: {sample.DisplayId}</Card.Title>
                <Card.Description>
                    Edit the details of the sample, including product variant,
                    location, state, owner, and mods.
                </Card.Description>
            </Card.Header>
            <Card.Content>
                <div class="flex flex-col gap-6">
                    <div>
                        <Label for="variant-select" class="mb-2"
                            >Product Variant</Label>
                        <ProductSelect
                            bind:bindValue={product_id}
                            placeholder="Select a product variant"
                            id="variant-select" />
                    </div>
                    <div>
                        <Label for="product-issue" class="mb-2"
                            >PCB Issue</Label>
                        <Input
                            type="text"
                            bind:value={product_issue}
                            placeholder="PCB Issue"
                            id="product-issue"
                            class="border rounded px-2 py-1 w-full" />
                    </div>
                    <div>
                        <Label for="location-select" class="mb-2"
                            >Location</Label>
                        <LocationSelect
                            bind:bindValue={location_id}
                            id="location-select" />
                    </div>
                    <div>
                        <Label for="state-select" class="mb-2">State</Label>
                        <StateSelect
                            bind:bindValue={sample_state}
                            id="state-select" />
                    </div>
                    <div>
                        <Label for="owner-select" class="mb-2">Owner</Label>
                        <UserSelect
                            bind:bindValue={owner_id}
                            placeholder="Select owner"
                            id="owner-select" />
                    </div>
                </div>
            </Card.Content>
        </Card.Root>

        <Card.Root class="flex-grow max-h-full overflow-y-auto">
            <Card.Header>
                <Card.Title>Mods</Card.Title>
                <Card.Description>
                    Manage mods associated with this sample. You can add,
                    remove, and view fitted mods.
                </Card.Description>
            </Card.Header>
            <Card.Content>
                <div class="flex flex-wrap gap-2 mb-2 items-center">
                    <Checkbox
                        bind:checked={showFittedOnly}
                        id="show-fitted-mods" />
                    <Label class="text-sm" for="show-fitted-mods"
                        >Show only fitted mods</Label>
                </div>
                {#if add_mod_error}
                    <div class="text-red-500 text-sm mb-2">{add_mod_error}</div>
                {/if}
                <div class="max-h-[30vh] overflow-y-auto">
                    <Table.Root>
                        <Table.Header>
                            <Table.Row>
                                <Table.Head>Name</Table.Head>
                                <Table.Head>Date Added</Table.Head>
                                <Table.Head>Date Removed</Table.Head>
                                <Table.Head>Status</Table.Head>
                                <Table.Head>Action</Table.Head>
                            </Table.Row>
                        </Table.Header>
                        <Table.Body>
                            {#if sample.mods.length === 0}
                                <Table.Row>
                                    <Table.Cell
                                        colspan={5}
                                        class="text-muted-foreground"
                                        >No mods.</Table.Cell>
                                </Table.Row>
                            {:else}
                                {#each [...sample.mods]
                                    .filter((mod) => !showFittedOnly || !mod.time_removed)
                                    .sort((a, b) => {
                                        // Sort by time_added descending, then by time_removed descending if both removed
                                        const aTime = a.time_removed ? a.time_removed.getTime() : (a.time_added?.getTime() ?? 0);
                                        const bTime = b.time_removed ? b.time_removed.getTime() : (b.time_added?.getTime() ?? 0);
                                        return bTime - aTime;
                                    }) as mod (mod.id)}
                                    <Table.Row
                                        class={!mod.time_removed
                                            ? 'font-bold bg-green-50 dark:bg-green-900/20'
                                            : ''}>
                                        <Table.Cell
                                            class="max-w-32 overflow-hidden text-ellipsis"
                                            >{mod.name}</Table.Cell>
                                        <Table.Cell
                                            >{mod.time_added?.toLocaleString?.() ??
                                                ''}</Table.Cell>
                                        <Table.Cell
                                            >{mod.time_removed
                                                ? mod.time_removed.toLocaleString()
                                                : '-'}</Table.Cell>
                                        <Table.Cell>
                                            {#if !mod.time_removed}
                                                <span
                                                    class="text-green-700 dark:text-green-400"
                                                    >Fitted</span>
                                            {:else}
                                                <span class="text-gray-500"
                                                    >Removed</span>
                                            {/if}
                                        </Table.Cell>
                                        <Table.Cell>
                                            {#if !mod.time_removed}
                                                <Button
                                                    size="sm"
                                                    variant="destructive"
                                                    onclick={() =>
                                                        removeMod(mod.id)}
                                                    >Remove</Button>
                                            {/if}
                                        </Table.Cell>
                                    </Table.Row>
                                {/each}
                            {/if}
                        </Table.Body>
                    </Table.Root>
                </div>

                <div class="mt-4 flex flex-row w-full gap-2">
                    <Input
                        type="text"
                        class="border rounded px-2 py-1"
                        placeholder="Add mod..."
                        bind:value={new_mod_name}
                        onkeydown={(e) => {
                            if (e.key === 'Enter') {
                                e.preventDefault();
                                addMod();
                            }
                        }}
                        disabled={adding_mod} />
                    <Button
                        onclick={addMod}
                        disabled={adding_mod || !new_mod_name.trim()}
                        >Add</Button>
                </div>
            </Card.Content>
        </Card.Root>
    </div>
    <Card.Root>
        <Card.Content>
            <div>
                <Button type="submit" class="w-full">Save Changes</Button>
            </div>
        </Card.Content>
    </Card.Root>
</form>
