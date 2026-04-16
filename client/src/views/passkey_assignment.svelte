<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { toast } from 'svelte-sonner';

    import { createPasskeyCredential } from '$lib/auth/webauthn';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';

    const dispatch = createEventDispatcher<{ completed: void }>();

    let { assignmentToken = '' } = $props<{ assignmentToken?: string }>();

    let token = $state('');
    let working = $state(false);

    $effect(() => {
        if (!token && assignmentToken) {
            token = assignmentToken;
        }
    });

    async function assignPasskey() {
        if (!token || working) {
            return;
        }

        working = true;
        try {
            const beginRes = await fetch('/api/auth/register/begin', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ assignment_token: token }),
            });
            if (!beginRes.ok) {
                throw new Error(
                    (await beginRes.json()).error ||
                        'Registration begin failed',
                );
            }
            const begin = await beginRes.json();

            const credential = await createPasskeyCredential(
                begin.challenge,
                begin.user_id,
                begin.user_name || begin.user_id,
            );

            const finishRes = await fetch('/api/auth/register/finish', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    challenge_token: begin.challenge_token,
                    challenge: begin.challenge,
                    credential_id: credential.credentialID,
                    public_key: credential.publicKey,
                    transports: credential.transports,
                    label: 'Sign-in passkey',
                }),
            });
            if (!finishRes.ok) {
                throw new Error(
                    (await finishRes.json()).error || 'Registration failed',
                );
            }

            toast.success('Passkey assigned successfully.');
            dispatch('completed');
        } catch (error) {
            const message =
                error instanceof Error ? error.message : String(error);
            toast.error(`Passkey assignment failed: ${message}`);
        } finally {
            working = false;
        }
    }
</script>

<div class="w-full h-full flex items-center justify-center p-4">
    <Card.Root class="w-full max-w-lg">
        <Card.Header>
            <Card.Title>Finish Account Setup</Card.Title>
            <Card.Description>
                Set up sign-in for this account.
            </Card.Description>
        </Card.Header>
        <Card.Content class="space-y-4">
            <Button
                class="w-full"
                onclick={assignPasskey}
                disabled={working || !token}>
                {working ? 'Setting Up Sign-In...' : 'Set Up Sign-In'}
            </Button>
        </Card.Content>
    </Card.Root>
</div>
