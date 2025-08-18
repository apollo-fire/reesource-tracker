<script lang="ts">
    import { AppStore } from '$lib/components/app_store';
    import type { SampleLocation } from '$lib/components/location';

    import * as Select from '../ui/select';
    import LocationTree from './LocationTree.svelte';

    let {
        bindValue = $bindable(''),
        disabled = false,
        placeholder = 'Select a location',
        id = 'location-select',
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

    let top_level_locations: SampleLocation[] = $state([]);

    AppStore.subscribe((new_val) => {
        top_level_locations =
            new_val.locations.filter(
                (l: SampleLocation) =>
                    (l.parentLocationID === '' ||
                        l.parentLocationID === null) &&
                    l.name !== '',
            ) ?? [];
    });
</script>

<Select.Root
    onValueChange={onValueChange}
    bind:value={bindValue}
    type="single"
    disabled={disabled}
    required={required}>
    <Select.Trigger id={id} class="w-full">
        {#if bindValue}
            {#if options.length > 0}
                {options.find((o) => o.value === bindValue)?.label}
            {:else if bindValue.startsWith('any-')}
                {(() => {
                    const parentId = bindValue.slice(4);
                    const parent = $AppStore.locations.find(
                        (l) => l.id === parentId,
                    );
                    return `Anywhere in ${parent ? parent.name : ''}`;
                })()}
            {:else}
                {$AppStore.locations.find((l) => l.id === bindValue)?.name}
            {/if}
        {:else}
            {placeholder}
        {/if}
    </Select.Trigger>
    <Select.Content class="w-full">
        <Select.Item value="">None</Select.Item>
        {#if options.length > 0}
            {#each options.filter((option) => !filterOutIds.includes(option.value)) as option (option.value)}
                <Select.Item value={option.value}>{option.label}</Select.Item>
            {/each}
        {:else}
            <LocationTree
                locations={top_level_locations}
                filterOutIds={filterOutIds}
                level={0}
                filterMode={filterMode} />
        {/if}
    </Select.Content>
</Select.Root>
