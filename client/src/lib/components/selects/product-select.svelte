<script lang="ts">
  import ProductTree from "./ProductTree.svelte";
  import * as Select from "../ui/select";
  import { AppStore } from "$lib/components/app_store";
  import type { SampleProduct } from "$lib/components/product";
  let {
    bindValue = $bindable(""),
    disabled = false,
    placeholder = "Select a product",
    id = "product-select",
    required = false,
    options = [],
    onValueChange = () => {},
    filterMode = false,
    filterOutIds = [],
  }: {
    bindValue?: string;
    disabled?: boolean;
    placeholder?: string;
    id?: string;
    required?: boolean;
    options?: { value: string; label: string }[];
    onValueChange?: (value: string) => void;
    filterMode?: boolean;
    filterOutIds?: string[];
  } = $props();

  let top_level_products: SampleProduct[] = $state([]);

  AppStore.subscribe((new_val) => {
    top_level_products =
      new_val.products.filter(
        (p: SampleProduct) =>
          (p.parentProductID === "" || p.parentProductID === null) &&
          p.name !== ""
      ) ?? [];
  });

  function findProductById(id: string): SampleProduct | null {
    return $AppStore.products.find((product) => product.id === id) || null;
  }
</script>

<Select.Root
  {onValueChange}
  bind:value={bindValue}
  type="single"
  {disabled}
  {required}
>
  <Select.Trigger {id} class="w-full">
    {#if bindValue}
      {#if options.length > 0}
        {options.find((o) => o.value === bindValue)?.label}
      {:else if bindValue.startsWith("any-")}
        {(() => {
          const parentId = bindValue.slice(4);
          const parent = findProductById(parentId);
          return `Any variant of ${parent ? parent.CombinedName : ""}`;
        })()}
      {:else}
        {findProductById(bindValue)?.CombinedName}
      {/if}
    {:else}
      {placeholder}
    {/if}
  </Select.Trigger>
  <Select.Content class="w-full">
    <Select.Item value="">None</Select.Item>
    {#if options.length > 0}
      {#each options.filter((option) => !filterOutIds.includes(option.value)) as option}
        <Select.Item value={option.value}>{option.label}</Select.Item>
      {/each}
    {:else}
      <ProductTree
        products={top_level_products}
        {filterOutIds}
        level={0}
        {filterMode}
      />
    {/if}
  </Select.Content>
</Select.Root>
