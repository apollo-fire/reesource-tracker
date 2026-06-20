<script lang="ts">
    import { UserCog } from 'lucide-svelte';
    import { createEventDispatcher } from 'svelte';

    import { Button } from '$lib/components/ui/button';
    import * as DropdownMenu from '$lib/components/ui/dropdown-menu';

    let {
        userDisplayName = 'Unknown User',
        userRolesDisplay = 'No roles',
        showManageRegistrations = false,
    } = $props<{
        userDisplayName?: string;
        userRolesDisplay?: string;
        showManageRegistrations?: boolean;
    }>();

    const dispatch = createEventDispatcher<{
        manageRegistrations: void;
        signOut: void;
    }>();

    function handleManageRegistrations() {
        dispatch('manageRegistrations');
    }

    function handleSignOut() {
        dispatch('signOut');
    }
</script>

<DropdownMenu.Root>
    <DropdownMenu.Trigger>
        {#snippet child({ props })}
            <Button
                {...props}
                type="button"
                variant="outline"
                class="whitespace-nowrap gap-2">
                <UserCog class="size-4" />
                {userDisplayName}
            </Button>
        {/snippet}
    </DropdownMenu.Trigger>
    <DropdownMenu.Content class="w-64" align="end">
        <DropdownMenu.Label>Signed In User</DropdownMenu.Label>
        <DropdownMenu.Group>
            <DropdownMenu.Item disabled>
                <span class="font-medium">{userDisplayName}</span>
            </DropdownMenu.Item>
            <DropdownMenu.Item disabled>
                <span class="text-muted-foreground">{userRolesDisplay}</span>
            </DropdownMenu.Item>
        </DropdownMenu.Group>
        <DropdownMenu.Separator />
        {#if showManageRegistrations}
            <DropdownMenu.Item onclick={handleManageRegistrations}>
                Manage My Registrations
            </DropdownMenu.Item>
        {/if}
        <DropdownMenu.Item onclick={handleSignOut}>Sign Out</DropdownMenu.Item>
    </DropdownMenu.Content>
</DropdownMenu.Root>
