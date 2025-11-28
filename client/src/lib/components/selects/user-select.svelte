<script lang="ts">
    import { AppStore } from '$lib/components/app_store';
    import type { User } from '$lib/components/user';

    import * as Select from '../ui/select';

    let {
        bindValue = $bindable(''),
        disabled = false,
        placeholder = 'Select a user',
        id = 'user-select',
        required = false,
        options = [],
        onValueChange = () => {},
        filterOutIds = [],
        filterMode = false,
    }: {
        bindValue?: string;
        disabled?: boolean;
        placeholder?: string;
        id?: string;
        required?: boolean;
        options?: { value: string; label: string }[];
        onValueChange?: (value: string) => void;
        filterOutIds?: string[];
        filterMode?: boolean;
    } = $props();

    let users: User[] = $state([]);

    AppStore.subscribe((new_val) => {
        users = new_val.users ?? [];
    });

    function findUserById(id: string): User | null {
        return users.find((user) => user.id === id) || null;
    }
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
            {:else}
                {findUserById(bindValue)?.name}
            {/if}
        {:else}
            {placeholder}
        {/if}
    </Select.Trigger>
    <Select.Content class="w-full">
        {#if filterMode}
            <Select.Item value="">Any</Select.Item>
        {:else}
            <Select.Item value="">None</Select.Item>
        {/if}
        {#if options.length > 0}
            {#each options.filter((option) => !filterOutIds.includes(option.value) && option.label !== '') as option (option.value)}
                <Select.Item value={option.value}>{option.label}</Select.Item>
            {/each}
        {:else}
            {#each users.filter((user) => !filterOutIds.includes(user.id) && user.name !== '') as user (user.id)}
                <Select.Item value={user.id}>{user.name}</Select.Item>
            {/each}
        {/if}
    </Select.Content>
</Select.Root>
