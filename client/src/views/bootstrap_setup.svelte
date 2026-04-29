<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import { toast } from 'svelte-sonner';

    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Input } from '$lib/components/ui/input';
    import * as Select from '$lib/components/ui/select';

    type BootstrapUserOption = {
        id: string;
        name: string;
    };

    const dispatch = createEventDispatcher<{
        tokenSelected: { assignmentToken: string };
    }>();

    let users: BootstrapUserOption[] = $state([]);
    let selectedUserId = $state('');
    let newUserName = $state('');
    let loading = $state(true);
    let working = $state(false);

    async function loadOptions() {
        loading = true;
        try {
            const res = await fetch('/api/auth/bootstrap-options');
            if (!res.ok) {
                throw new Error(
                    (await res.json()).error || 'Failed to load setup options',
                );
            }
            const data = await res.json();

            if (data.assignment_token) {
                dispatch('tokenSelected', {
                    assignmentToken: data.assignment_token,
                });
                return;
            }

            users = Array.isArray(data.users) ? data.users : [];
            if (!selectedUserId && users.length > 0) {
                selectedUserId = users[0].id;
            }
        } catch (error) {
            toast.error(`Unable to load setup options: ${error}`);
        } finally {
            loading = false;
        }
    }

    async function selectExistingUser() {
        if (!selectedUserId || working) {
            return;
        }
        working = true;
        try {
            const res = await fetch('/api/auth/bootstrap/select-user', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ user_id: selectedUserId }),
            });
            if (!res.ok) {
                throw new Error(
                    (await res.json()).error || 'Failed to start setup',
                );
            }
            const data = await res.json();
            dispatch('tokenSelected', {
                assignmentToken: data.assignment_token,
            });
        } catch (error) {
            toast.error(`Setup failed: ${error}`);
        } finally {
            working = false;
        }
    }

    async function createUserAndContinue() {
        if (working) {
            return;
        }
        working = true;
        try {
            const res = await fetch('/api/auth/bootstrap/create-user', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: newUserName.trim() }),
            });
            if (!res.ok) {
                throw new Error(
                    (await res.json()).error || 'Failed to create account',
                );
            }
            const data = await res.json();
            dispatch('tokenSelected', {
                assignmentToken: data.assignment_token,
            });
        } catch (error) {
            toast.error(`Could not create account: ${error}`);
        } finally {
            working = false;
        }
    }

    onMount(loadOptions);
</script>

<div class="w-full h-full flex items-center justify-center p-4">
    <Card.Root class="w-full max-w-xl">
        <Card.Header>
            <Card.Title>Set Up First Admin</Card.Title>
            <Card.Description>
                Choose who should become the first admin account.
            </Card.Description>
        </Card.Header>
        <Card.Content class="space-y-6">
            {#if loading}
                <p class="text-sm text-muted-foreground"
                    >Loading setup options...</p>
            {:else}
                <div class="space-y-3">
                    <h3 class="text-sm font-semibold">Use Existing Account</h3>
                    {#if users.length === 0}
                        <p class="text-sm text-muted-foreground">
                            No existing accounts are available.
                        </p>
                    {:else}
                        <div class="flex flex-col gap-2 sm:flex-row">
                            <Select.Root
                                type="single"
                                value={selectedUserId}
                                onValueChange={(value) =>
                                    (selectedUserId = value)}>
                                <Select.Trigger class="w-full sm:w-80">
                                    {users.find(
                                        (user) => user.id === selectedUserId,
                                    )?.name || 'Select a user'}
                                </Select.Trigger>
                                <Select.Content>
                                    {#each users as user (user.id)}
                                        <Select.Item value={user.id}
                                            >{user.name}</Select.Item>
                                    {/each}
                                </Select.Content>
                            </Select.Root>
                            <Button
                                class="sm:min-w-40"
                                onclick={selectExistingUser}
                                disabled={working || !selectedUserId}
                                >{working ? 'Starting...' : 'Continue'}</Button>
                        </div>
                    {/if}
                </div>

                <div class="space-y-3">
                    <h3 class="text-sm font-semibold"
                        >Create New Admin Account</h3>
                    <div class="flex flex-col gap-2 sm:flex-row">
                        <Input
                            placeholder="Name (optional)"
                            bind:value={newUserName}
                            class="sm:w-80" />
                        <Button
                            variant="outline"
                            class="sm:min-w-40"
                            onclick={createUserAndContinue}
                            disabled={working}
                            >{working
                                ? 'Creating...'
                                : 'Create & Continue'}</Button>
                    </div>
                </div>
            {/if}
        </Card.Content>
    </Card.Root>
</div>
