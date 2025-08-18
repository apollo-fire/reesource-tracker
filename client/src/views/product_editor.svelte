<script lang="ts">
    import { toast } from 'svelte-sonner';
    import { SvelteMap } from 'svelte/reactivity';

    import { AppStore } from '$lib/components/app_store';
    import { SampleProduct } from '$lib/components/product';
    import ProductSelect from '$lib/components/selects/product-select.svelte';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Input } from '$lib/components/ui/input';
    import {
        Table,
        TableBody,
        TableCell,
        TableHead,
        TableHeader,
        TableRow,
    } from '$lib/components/ui/table';

    let updateTimeouts: SvelteMap<
        string,
        ReturnType<typeof setTimeout>
    > = new SvelteMap();
    // Store for parent product IDs, since it needs a string value for Select
    let parent_product_id: { [key: string]: string } = $state({});

    AppStore.subscribe((data) => {
        data.products.forEach((product) => {
            if (product.parentProductID) {
                parent_product_id[product.id] = product.parentProductID;
            } else {
                parent_product_id[product.id] = '';
            }
        });
    });

    async function updateProduct(product: SampleProduct) {
        let payload = {
            name: product.name,
            parent_product_id: parent_product_id[product.id] || null,
            part_number: product.partNumber || '',
        };
        const res = await fetch(`/api/product/${product.id}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        if (!res.ok) {
            toast.error('Failed to update product.');
            throw new Error(await res.text());
        }
        toast.success('Product updated successfully.');
    }

    // Debounced update function
    function debounceUpdate(product: SampleProduct) {
        if (updateTimeouts.has(product.id)) {
            clearTimeout(updateTimeouts.get(product.id));
        }
        const timeout = setTimeout(() => {
            updateProduct(product);
            updateTimeouts.delete(product.id);
        }, 1000);
        updateTimeouts.set(product.id, timeout);
    }

    // Add a new product row by creating it on the backend
    async function addProductRow() {
        await fetch('/api/product', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: '', parent_product_id: null }),
        });
    }

    async function deleteProduct(product: SampleProduct) {
        // If the product has children, we cannot delete it
        if (product.ChildProducts.length > 0) {
            toast.error('Cannot delete product with child products.');
            return;
        }
        await fetch(`/api/product/${product.id}`, { method: 'DELETE' });
    }
</script>

<Card.Root class="h-full max-h-full overflow-y-auto max-h-[calc(100vh-6rem)]">
    <Card.Header>
        <Card.Title>Product Editor</Card.Title>
        <Card.Description>
            Edit products and their hierarchy. Changes are saved automatically.
        </Card.Description>
    </Card.Header>
    <Card.Content class="flex flex-col h-full overflow-hidden">
        <div class="flex-1 min-h-0 flex-basis-0 flex-shrink min-h-0">
            <div class="w-full h-full max-h-full overflow-y-auto">
                <Table class="w-full flex-grow">
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Part Number</TableHead>
                            <TableHead>Parent Product</TableHead>
                            <TableHead>Delete</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {#each $AppStore.products as product, i (product.id || i)}
                            <TableRow>
                                <TableCell>
                                    <Input
                                        class=" w-full my-2"
                                        bind:value={product.name}
                                        oninput={(e: Event) => {
                                            const target =
                                                e.target as HTMLInputElement;
                                            product.name = target.value;
                                            debounceUpdate(product);
                                        }}
                                        placeholder="Product Name" />
                                </TableCell>
                                <TableCell>
                                    <Input
                                        class=" w-full my-2"
                                        bind:value={product.partNumber}
                                        oninput={(e: Event) => {
                                            const target =
                                                e.target as HTMLInputElement;
                                            product.partNumber = target.value;
                                            debounceUpdate(product);
                                        }}
                                        placeholder="Part Number" />
                                </TableCell>
                                <TableCell>
                                    <ProductSelect
                                        bind:bindValue={
                                            parent_product_id[product.id]
                                        }
                                        onValueChange={(v) => {
                                            parent_product_id[product.id] = v;
                                            parent_product_id =
                                                parent_product_id;
                                            debounceUpdate(product);
                                        }}
                                        placeholder="None"
                                        id={`parent-product-${product.id}`}
                                        disabled={false}
                                        required={false}
                                        filterOutIds={[
                                            product.id,
                                            ...product.ChildProducts.map(
                                                (child) => child.id,
                                            ),
                                        ]}
                                        options={$AppStore.products
                                            .filter(
                                                (p) =>
                                                    p.id !== product.id &&
                                                    !product.ChildProducts.some(
                                                        (child) =>
                                                            child.id === p.id,
                                                    ),
                                            )
                                            .map((p) => ({
                                                value: p.id,
                                                label: p.name,
                                            }))} />
                                </TableCell>
                                <TableCell>
                                    <Button
                                        type="button"
                                        variant="destructive"
                                        onclick={() => deleteProduct(product)}>
                                        Delete
                                    </Button>
                                </TableCell>
                            </TableRow>
                        {/each}
                    </TableBody>
                </Table>
            </div>
        </div>
        <div class="mt-4 w-full flex flex-row self-end justify-end">
            <Button type="button" onclick={addProductRow}>Add Product</Button>
        </div>
    </Card.Content>
</Card.Root>
