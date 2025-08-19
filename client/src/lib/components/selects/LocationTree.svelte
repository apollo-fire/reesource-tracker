<script lang="ts">
    import type { SampleLocation } from '$lib/components/location';

    import * as Select from '../ui/select';
    import LocationTree from './LocationTree.svelte';

    let {
        locations = [],
        filterOutIds = [],
        level = 0,
        filterMode = false,
    }: {
        locations: SampleLocation[];
        filterOutIds: string[];
        level: number;
        filterMode: boolean;
    } = $props();
</script>

{#each locations as location (location.id)}
    {#if !filterOutIds.includes(location.id) && location.name !== ''}
        {#if location.ChildLocations && location.ChildLocations.length > 0}
            <Select.Label style="padding-left: max({level * 1.5}em, 0.5em)"
                >{location.CombinedName}</Select.Label>
            {#if !filterOutIds.includes(`any-${location.id}`) && location.ChildLocations.length > 1 && filterMode}
                <Select.Item
                    value={'any-' + location.id}
                    style="padding-left: {(level + 1) * 1.5}em">
                    Anywhere in {location.CombinedName}
                </Select.Item>
            {/if}
            <LocationTree
                locations={location.ChildLocations}
                filterOutIds={filterOutIds}
                level={level + 1}
                filterMode={filterMode} />
        {:else}
            <Select.Item
                value={location.id}
                style="padding-left: {level * 1.5}em">
                {location.CombinedName}
            </Select.Item>
        {/if}
    {/if}
{/each}
