<script lang="ts">
    import { onMount } from 'svelte';
    import { toast } from 'svelte-sonner';
    import { SvelteMap } from 'svelte/reactivity';

    import {
        ROLE_OPTIONS,
        canManageRegistrations as canManageRegistrationsForUser,
        formatRoleLabel,
        getPrimaryRole,
    } from '$lib/auth/user_management';
    import { AppStore, UpdateUsers } from '$lib/components/app_store';
    import { Base64UUIDToString } from '$lib/components/id_helper';
    import { SampleState } from '$lib/components/sample';
    import * as AlertDialog from '$lib/components/ui/alert-dialog';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Input } from '$lib/components/ui/input';
    import * as Select from '$lib/components/ui/select';
    import {
        Table,
        TableBody,
        TableCell,
        TableHead,
        TableHeader,
        TableRow,
    } from '$lib/components/ui/table';
    import type { User } from '$lib/components/user';
    import ManageRegistrationsDialog from '$lib/components/user/manage_registrations_dialog.svelte';

    let isAdmin = $state(false);
    let currentUserId = $state('');
    let confirmOpen = $state(false);
    let confirmTitle = $state('Confirm Action');
    let confirmMessage = $state('');
    let confirmBusy = $state(false);
    let onConfirmAction: (() => Promise<void>) | null = null;
    let manageRegistrationsOpen = $state(false);
    let selectedRegistrationUserId = $state('');
    let selectedRegistrationUserLabel = $state('');

    function openConfirm(
        title: string,
        message: string,
        action: () => Promise<void>,
    ) {
        confirmTitle = title;
        confirmMessage = message;
        onConfirmAction = action;
        confirmOpen = true;
    }

    async function runConfirmedAction() {
        if (!onConfirmAction || confirmBusy) {
            return;
        }
        confirmBusy = true;
        try {
            await onConfirmAction();
            confirmOpen = false;
        } finally {
            confirmBusy = false;
        }
    }

    async function refreshSession() {
        const res = await fetch('/api/auth/session');
        if (!res.ok) {
            isAdmin = false;
            currentUserId = '';
            return;
        }
        const data = await res.json();
        isAdmin =
            !!data.authenticated &&
            Array.isArray(data.roles) &&
            data.roles.includes('admin');
        currentUserId =
            typeof data?.user?.ID === 'string'
                ? Base64UUIDToString(data.user.ID)
                : '';
    }

    function canManageRegistrations(user: User): boolean {
        return canManageRegistrationsForUser(isAdmin, currentUserId, user.id);
    }

    let updateTimeouts: SvelteMap<
        string,
        ReturnType<typeof setTimeout>
    > = new SvelteMap();

    async function updateUser(user: User) {
        const payload = { name: user.name };
        const res = await fetch(`/api/user/${user.id}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        if (!res.ok) {
            toast.error('Failed to update user.');
            throw new Error(await res.text());
        }
        toast.success('User updated successfully.');
    }

    function debounceUpdate(user: User) {
        if (updateTimeouts.has(user.id)) {
            clearTimeout(updateTimeouts.get(user.id));
        }
        const timeout = setTimeout(() => {
            updateUser(user);
            updateTimeouts.delete(user.id);
        }, 1000);
        updateTimeouts.set(user.id, timeout);
    }

    async function addUserRow() {
        const res = await fetch('/api/user', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: '' }),
        });
        if (!res.ok) {
            toast.error('Failed to add user.');
            return;
        }
        toast.success('User added.');
    }

    async function deleteUser(user: User) {
        openConfirm(
            'Delete User',
            `Delete ${user.name || user.id}? This cannot be undone.`,
            async () => {
                const res = await fetch(`/api/user/${user.id}`, {
                    method: 'DELETE',
                    headers: { 'X-Confirm-Action': 'confirm' },
                });
                if (!res.ok) {
                    toast.error('Failed to delete user.');
                    return;
                }
                toast.success('User deleted.');
            },
        );
    }

    async function updateRoleSelection(user: User, selectedRole: string) {
        if (!isAdmin) {
            return;
        }

        if (!['admin', 'maintainer', 'user'].includes(selectedRole)) {
            toast.error('Invalid role selection.');
            return;
        }

        const currentRole = getPrimaryRole(user.roles);
        if (currentRole === selectedRole) {
            return;
        }

        openConfirm(
            'Change Role',
            `Change ${user.name || user.id} from ${currentRole} to ${selectedRole}?`,
            async () => {
                for (const role of ROLE_OPTIONS) {
                    const hasRole = user.roles.includes(role);
                    if (role === selectedRole && !hasRole) {
                        const addRes = await fetch(
                            `/api/user/${user.id}/roles`,
                            {
                                method: 'POST',
                                headers: {
                                    'Content-Type': 'application/json',
                                    'X-Confirm-Action': 'confirm',
                                },
                                body: JSON.stringify({ role }),
                            },
                        );
                        if (!addRes.ok) {
                            toast.error(`Failed to assign ${selectedRole}.`);
                            return;
                        }
                    }

                    if (role !== selectedRole && hasRole) {
                        const removeRes = await fetch(
                            `/api/user/${user.id}/roles/${role}`,
                            {
                                method: 'DELETE',
                                headers: {
                                    'X-Confirm-Action': 'confirm',
                                },
                            },
                        );
                        if (!removeRes.ok) {
                            toast.error(`Failed to remove ${role}.`);
                            return;
                        }
                    }
                }

                await UpdateUsers();
                await refreshSession();
                toast.success('Role updated.');
            },
        );
    }

    function openManageRegistrations(user: User) {
        selectedRegistrationUserId = user.id;
        selectedRegistrationUserLabel = user.name || user.id;
        manageRegistrationsOpen = true;
    }

    onMount(async () => {
        await refreshSession();
    });

    $effect(() => {
        if (!manageRegistrationsOpen) {
            selectedRegistrationUserId = '';
            selectedRegistrationUserLabel = '';
        }
    });

    const valid_states = Object.values(SampleState).filter(
        (a) => a !== SampleState.unknown,
    );
</script>

<AlertDialog.Root bind:open={confirmOpen}>
    <AlertDialog.Content>
        <AlertDialog.Header>
            <AlertDialog.Title>{confirmTitle}</AlertDialog.Title>
            <AlertDialog.Description>{confirmMessage}</AlertDialog.Description>
        </AlertDialog.Header>
        <AlertDialog.Footer>
            <AlertDialog.Cancel disabled={confirmBusy}
                >Cancel</AlertDialog.Cancel>
            <AlertDialog.Action
                variant="destructive"
                disabled={confirmBusy}
                onclick={runConfirmedAction}
                >{confirmBusy ? 'Working...' : 'Confirm'}</AlertDialog.Action>
        </AlertDialog.Footer>
    </AlertDialog.Content>
</AlertDialog.Root>

<ManageRegistrationsDialog
    bind:open={manageRegistrationsOpen}
    userId={selectedRegistrationUserId}
    userLabel={selectedRegistrationUserLabel}
    useAdminEndpoints={isAdmin} />

<Card.Root class="h-full max-h-full overflow-y-auto max-h-[calc(100vh-6rem)]">
    <Card.Header>
        <Card.Title>User Editor</Card.Title>
        <Card.Description>
            Edit users. Changes are saved automatically.
        </Card.Description>
    </Card.Header>
    <Card.Content class="flex flex-col h-full overflow-hidden">
        <div class="flex-1 min-h-0 flex-basis-0 flex-shrink min-h-0">
            <div class="w-full h-full max-h-full overflow-y-auto">
                <Table class="w-full flex-grow">
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Roles</TableHead>
                            {#each valid_states as state_type (state_type)}
                                <TableHead class="capitalise"
                                    >Samples {state_type.replace(
                                        '_',
                                        ' ',
                                    )}</TableHead>
                            {/each}
                            <TableHead></TableHead>
                            <TableHead></TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {#each $AppStore.users as user, i (user.id || i)}
                            <TableRow>
                                <TableCell>
                                    <Input
                                        class="input input-bordered w-full my-2"
                                        bind:value={user.name}
                                        oninput={(e: Event) => {
                                            const target =
                                                e.target as HTMLInputElement;
                                            user.name = target.value;
                                            debounceUpdate(user);
                                        }}
                                        placeholder="User Name" />
                                </TableCell>
                                <TableCell>
                                    <div
                                        class="flex flex-wrap gap-2 items-center">
                                        {#if isAdmin}
                                            <Select.Root
                                                type="single"
                                                value={getPrimaryRole(
                                                    user.roles,
                                                )}
                                                onValueChange={(value) =>
                                                    updateRoleSelection(
                                                        user,
                                                        value,
                                                    )}>
                                                <Select.Trigger class="w-40">
                                                    {formatRoleLabel(
                                                        getPrimaryRole(
                                                            user.roles,
                                                        ),
                                                    )}
                                                </Select.Trigger>
                                                <Select.Content>
                                                    {#each ROLE_OPTIONS as role (role)}
                                                        <Select.Item
                                                            value={role}
                                                            >{formatRoleLabel(
                                                                role,
                                                            )}</Select.Item>
                                                    {/each}
                                                </Select.Content>
                                            </Select.Root>
                                        {:else}
                                            <span
                                                class="text-xs text-muted-foreground">
                                                {formatRoleLabel(
                                                    getPrimaryRole(user.roles),
                                                )}
                                            </span>
                                        {/if}
                                    </div>
                                </TableCell>

                                {#each valid_states as state_type (state_type)}
                                    <TableCell>
                                        {user.AssignedSamples[state_type] || 0}
                                    </TableCell>
                                {/each}
                                <TableCell>
                                    {#if canManageRegistrations(user)}
                                        <Button
                                            type="button"
                                            variant="outline"
                                            onclick={() =>
                                                openManageRegistrations(user)}>
                                            Manage Registrations
                                        </Button>
                                    {/if}
                                </TableCell>
                                <TableCell>
                                    <Button
                                        type="button"
                                        variant="destructive"
                                        onclick={() => deleteUser(user)}>
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
            <Button type="button" onclick={addUserRow}>Add User</Button>
        </div>
    </Card.Content>
</Card.Root>
