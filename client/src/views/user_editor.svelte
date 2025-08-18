<script lang="ts">
    import { toast } from 'svelte-sonner';

    import { AppStore } from '$lib/components/app_store';
    import { SampleState } from '$lib/components/sample';
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
    import type { User } from '$lib/components/user';

    let updateTimeouts = new Map<string, ReturnType<typeof setTimeout>>();
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
        console.log(user);
        const res = await fetch(`/api/user/${user.id}`, { method: 'DELETE' });
        if (!res.ok) {
            toast.error('Failed to delete user.');
            return;
        }
        toast.success('User deleted.');
    }

    const valid_states = Object.values(SampleState).filter(
        (a) => a !== SampleState.unknown,
    );
</script>

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

                                {#each valid_states as state_type (state_type)}
                                    <TableCell>
                                        {user.AssignedSamples[state_type] || 0}
                                    </TableCell>
                                {/each}
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
