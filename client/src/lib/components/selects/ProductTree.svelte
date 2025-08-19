<script lang="ts">
    import type { SampleProduct } from '$lib/components/product';
    import * as Select from '$lib/components/ui/select';

    import ProductTree from './ProductTree.svelte';

    let {
        products,
        filterOutIds = [],
        level = 0,
        filterMode = false,
    }: {
        products: SampleProduct[];
        filterOutIds: string[];
        level: number;
        filterMode: boolean;
    } = $props();
</script>

{#each products as product (product.id)}
    {#if !filterOutIds.includes(product.id) && product.name !== ''}
        {#if product.ChildProducts && product.ChildProducts.length > 0}
            <Select.Label style="padding-left: max({level * 1.5}em, 0.5em)"
                >{product.CombinedName}</Select.Label>
            {#if !filterOutIds.includes(`any-${product.id}`) && product.ChildProducts.length > 1 && filterMode}
                <Select.Item
                    value={'any-' + product.id}
                    style="padding-left: {(level + 1) * 1.5}em">
                    Any variant of {product.CombinedName}
                </Select.Item>
            {/if}
            <ProductTree
                products={product.ChildProducts}
                filterOutIds={filterOutIds}
                level={level + 1}
                filterMode={filterMode} />
        {:else}
            <Select.Item
                value={product.id}
                style="padding-left: {level * 1.5}em"
                >{product.CombinedName}
                {#if product.partNumber}
                    ({product.partNumber}){/if}</Select.Item>
        {/if}
    {/if}
{/each}
