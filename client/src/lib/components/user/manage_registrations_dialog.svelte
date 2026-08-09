<script lang="ts">
    import { toast } from 'svelte-sonner';

    import * as AlertDialog from '$lib/components/ui/alert-dialog';
    import { Button } from '$lib/components/ui/button';
    import * as Dialog from '$lib/components/ui/dialog/index.js';
    import { Input } from '$lib/components/ui/input';

    type RegistrationLink = {
        linkId: number;
        url: string;
        expiresAt: string;
    };

    type RegisteredPasskey = {
        credentialId: string;
        label: string;
        createdAt: string;
        isCurrentSession: boolean;
    };

    let {
        open = $bindable(false),
        userId = '',
        userLabel = 'User',
        useAdminEndpoints = false,
    } = $props<{
        open?: boolean;
        userId?: string;
        userLabel?: string;
        useAdminEndpoints?: boolean;
    }>();

    let registrationLink = $state<RegistrationLink | null>(null);
    let registeredPasskeys = $state<RegisteredPasskey[]>([]);

    let confirmOpen = $state(false);
    let confirmTitle = $state('Confirm Action');
    let confirmMessage = $state('');
    let confirmBusy = $state(false);
    let onConfirmAction: (() => Promise<void>) | null = null;

    let loadedState = $state('');

    function hasTargetUser(): boolean {
        return userId.trim().length > 0;
    }

    function assignmentLinkEndpoint(): string {
        return useAdminEndpoints
            ? `/api/auth/admin/users/${userId}/assignment-link`
            : '/api/auth/self/assignment-link';
    }

    function passkeysEndpoint(): string {
        return useAdminEndpoints
            ? `/api/auth/admin/users/${userId}/passkeys`
            : '/api/auth/self/passkeys';
    }

    function revokePasskeyEndpoint(credentialId: string): string {
        return useAdminEndpoints
            ? `/api/auth/admin/passkeys/${credentialId}/revoke`
            : `/api/auth/self/passkeys/${credentialId}/revoke`;
    }

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

    async function loadRegistrationLink() {
        if (!hasTargetUser()) {
            return;
        }

        const res = await fetch(assignmentLinkEndpoint());
        if (!res.ok) {
            return;
        }

        const data = await res.json();
        if (!data.has_active_link) {
            registrationLink = null;
            return;
        }

        registrationLink = {
            linkId: Number(data.link_id) || 0,
            url:
                typeof data.assignment_url === 'string'
                    ? data.assignment_url
                    : '',
            expiresAt:
                typeof data.expires_at === 'string' ? data.expires_at : '',
        };
    }

    async function loadRegisteredPasskeys() {
        if (!hasTargetUser()) {
            return;
        }

        const res = await fetch(passkeysEndpoint());
        if (!res.ok) {
            return;
        }

        const data = await res.json();
        registeredPasskeys = Array.isArray(data)
            ? data
                  .filter((p) => typeof p.credential_id === 'string')
                  .map((p) => ({
                      credentialId: p.credential_id,
                      label:
                          typeof p.label === 'string' && p.label.trim()
                              ? p.label
                              : 'Passkey',
                      createdAt:
                          typeof p.created_at === 'string' ? p.created_at : '',
                      isCurrentSession: Boolean(p.is_current_session),
                  }))
            : [];
    }

    async function createRegistrationLink() {
        if (!hasTargetUser()) {
            return;
        }

        const res = await fetch(assignmentLinkEndpoint(), {
            method: 'POST',
        });
        if (!res.ok) {
            toast.error('Failed to create registration link.');
            return;
        }

        const data = await res.json();
        registrationLink = {
            linkId: Number(data.link_id) || 0,
            url:
                typeof data.assignment_url === 'string'
                    ? data.assignment_url
                    : '',
            expiresAt:
                typeof data.expires_at === 'string' ? data.expires_at : '',
        };
        toast.success('Registration link created.');
    }

    async function deleteRegistrationLink() {
        if (!hasTargetUser()) {
            return;
        }

        openConfirm(
            'Delete Registration Link',
            `Delete the active registration link for ${userLabel || userId}?`,
            async () => {
                const res = await fetch(assignmentLinkEndpoint(), {
                    method: 'DELETE',
                    headers: { 'X-Confirm-Action': 'confirm' },
                });
                if (!res.ok) {
                    toast.error('Failed to delete registration link.');
                    return;
                }
                registrationLink = null;
                toast.success('Registration link deleted.');
            },
        );
    }

    async function copyRegistrationLink(url: string) {
        try {
            await navigator.clipboard.writeText(url);
            toast.success('Registration link copied.');
        } catch {
            toast.error('Failed to copy registration link.');
        }
    }

    async function revokePasskeyRegistration(credentialId: string) {
        if (!hasTargetUser()) {
            return;
        }

        openConfirm(
            'Remove Passkey Registration',
            `Remove this passkey registration for ${userLabel || userId}?`,
            async () => {
                const res = await fetch(revokePasskeyEndpoint(credentialId), {
                    method: 'POST',
                    headers: { 'X-Confirm-Action': 'confirm' },
                });
                if (!res.ok) {
                    toast.error('Failed to remove passkey registration.');
                    return;
                }
                await loadRegisteredPasskeys();
                toast.success('Passkey registration removed.');
            },
        );
    }

    function formatPasskeyCreatedAt(createdAt: string): string {
        if (!createdAt) {
            return 'Created date unavailable';
        }
        const dt = new Date(createdAt);
        if (Number.isNaN(dt.getTime())) {
            return 'Created date unavailable';
        }
        return dt.toLocaleString();
    }

    function formatExpiry(expiresAt: string): string {
        if (!expiresAt) {
            return 'No expiration';
        }
        const dt = new Date(expiresAt);
        if (Number.isNaN(dt.getTime())) {
            return 'Expiration unavailable';
        }
        return dt.toLocaleString();
    }

    $effect(() => {
        if (!open) {
            return;
        }

        const state = [userId, userLabel, String(useAdminEndpoints)].join('|');
        if (state === loadedState) {
            return;
        }

        loadedState = state;
        void loadRegistrationLink();
        void loadRegisteredPasskeys();
    });

    $effect(() => {
        if (open) {
            return;
        }

        loadedState = '';
        registrationLink = null;
        registeredPasskeys = [];
    });
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

<Dialog.Root bind:open={open}>
    <Dialog.Content class="sm:max-w-2xl max-h-[85vh] overflow-y-auto">
        <Dialog.Header>
            <Dialog.Title>Manage Registrations</Dialog.Title>
            <Dialog.Description>
                Manage registration links and passkeys for
                {userLabel || userId}.
            </Dialog.Description>
        </Dialog.Header>

        <div class="space-y-4">
            <div class="space-y-2 border rounded-md p-3">
                <div class="text-sm font-medium">Registration Link</div>
                <Button
                    type="button"
                    variant="outline"
                    onclick={createRegistrationLink}>
                    {registrationLink
                        ? 'Replace Registration Link'
                        : 'Create Registration Link'}
                </Button>

                {#if registrationLink}
                    {@const link = registrationLink}
                    <Input readonly value={link.url} />
                    <div class="flex gap-2 flex-wrap">
                        <Button
                            type="button"
                            variant="secondary"
                            onclick={() => copyRegistrationLink(link.url)}>
                            Copy Link
                        </Button>
                        <Button
                            type="button"
                            variant="destructive"
                            onclick={deleteRegistrationLink}>
                            Delete Link
                        </Button>
                    </div>
                    <div class="text-xs text-muted-foreground">
                        Expires: {formatExpiry(link.expiresAt)}
                    </div>
                {:else}
                    <div class="text-xs text-muted-foreground">
                        No active registration link.
                    </div>
                {/if}
            </div>

            <div class="space-y-2 border rounded-md p-3">
                <div class="text-sm font-medium">Registered Passkeys</div>
                {#if registeredPasskeys.length}
                    {#each registeredPasskeys as passkey (passkey.credentialId)}
                        <div class="border rounded p-2 text-xs space-y-1">
                            <div class="font-medium">{passkey.label}</div>
                            <div class="break-all text-muted-foreground">
                                {passkey.credentialId}
                            </div>
                            <div class="text-muted-foreground">
                                Created: {formatPasskeyCreatedAt(
                                    passkey.createdAt,
                                )}
                            </div>
                            <Button
                                type="button"
                                size="sm"
                                variant="destructive"
                                disabled={passkey.isCurrentSession}
                                onclick={() =>
                                    revokePasskeyRegistration(
                                        passkey.credentialId,
                                    )}>
                                {passkey.isCurrentSession
                                    ? 'Current Session'
                                    : 'Remove Registration'}
                            </Button>
                        </div>
                    {/each}
                {:else}
                    <div class="text-xs text-muted-foreground">
                        No active passkey registrations.
                    </div>
                {/if}
            </div>
        </div>
    </Dialog.Content>
</Dialog.Root>
