<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { toast } from 'svelte-sonner';

    import { createPasskeyCredential } from '$lib/auth/webauthn';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Input } from '$lib/components/ui/input';

    const dispatch = createEventDispatcher<{ completed: void }>();

    let { assignmentToken = '' } = $props<{ assignmentToken?: string }>();

    let token = $state('');
    let working = $state(false);
    // 'choose' | 'passkey' | 'email'
    let mode = $state<'choose' | 'passkey' | 'email'>('choose');
    let email = $state('');

    $effect(() => {
        if (!token && assignmentToken) {
            token = assignmentToken;
        }
    });

    async function assignPasskey() {
        if (!token || working) return;
        working = true;
        try {
            const beginRes = await fetch('/api/auth/register/begin', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ assignment_token: token }),
            });
            if (!beginRes.ok) {
                throw new Error(
                    (await beginRes.json()).error || 'Registration begin failed',
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
                    client_data_json: credential.clientDataJSON,
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

    async function assignEmail() {
        if (!token || !email.trim() || working) return;
        working = true;
        try {
            const res = await fetch('/api/auth/email/register', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    assignment_token: token,
                    email: email.trim(),
                }),
            });
            if (!res.ok) {
                throw new Error(
                    (await res.json()).error || 'Email registration failed',
                );
            }
            toast.success('Email address registered. Use it to sign in with a magic link.');
            dispatch('completed');
        } catch (error) {
            const message =
                error instanceof Error ? error.message : String(error);
            toast.error(`Email registration failed: ${message}`);
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
                Choose how you want to sign in.
            </Card.Description>
        </Card.Header>
        <Card.Content class="space-y-4">
            {#if mode === 'choose'}
                <Button
                    class="w-full"
                    onclick={() => (mode = 'passkey')}
                    disabled={working || !token}>
                    Set Up Passkey
                </Button>
                <Button
                    variant="outline"
                    class="w-full"
                    onclick={() => (mode = 'email')}
                    disabled={working || !token}>
                    Register Email Address
                </Button>
            {:else if mode === 'passkey'}
                <p class="text-sm text-muted-foreground">
                    Your browser will prompt you to create a passkey for this
                    device.
                </p>
                <Button
                    class="w-full"
                    onclick={assignPasskey}
                    disabled={working || !token}>
                    {working ? 'Setting Up...' : 'Create Passkey'}
                </Button>
                <Button
                    variant="ghost"
                    class="w-full"
                    onclick={() => (mode = 'choose')}
                    disabled={working}>
                    Back
                </Button>
            {:else}
                <p class="text-sm text-muted-foreground">
                    Enter an email address. You will receive a sign-in link each
                    time you want to log in.
                </p>
                <Input
                    type="email"
                    placeholder="you@example.com"
                    bind:value={email}
                    disabled={working}
                    onkeydown={(e) => {
                        if (e.key === 'Enter') assignEmail();
                    }} />
                <Button
                    class="w-full"
                    onclick={assignEmail}
                    disabled={working || !email.trim() || !token}>
                    {working ? 'Registering...' : 'Register Email'}
                </Button>
                <Button
                    variant="ghost"
                    class="w-full"
                    onclick={() => (mode = 'choose')}
                    disabled={working}>
                    Back
                </Button>
            {/if}
        </Card.Content>
    </Card.Root>
</div>
