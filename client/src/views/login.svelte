<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import { toast } from 'svelte-sonner';

    import { getPasskeyAssertion } from '$lib/auth/webauthn';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Input } from '$lib/components/ui/input';

    const dispatch = createEventDispatcher<{ authenticated: void }>();

    let working = $state(false);
    let magicLinkEnabled = $state(false);
    let emailMode = $state(false);
    let email = $state('');
    let emailSent = $state(false);

    onMount(async () => {
        try {
            const res = await fetch('/api/auth/features');
            if (res.ok) {
                const data = await res.json();
                magicLinkEnabled = !!data.magic_links_enabled;
            }
        } catch {
            // passkey-only mode if features endpoint unavailable
        }
    });

    async function signIn() {
        if (working) {
            return;
        }

        working = true;
        try {
            const beginRes = await fetch('/api/auth/login/begin', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({}),
            });
            if (!beginRes.ok) {
                throw new Error(
                    (await beginRes.json()).error || 'Login begin failed',
                );
            }
            const begin = await beginRes.json();

            const assertion = await getPasskeyAssertion(begin.challenge);

            const finishRes = await fetch('/api/auth/login/finish', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    challenge_token: begin.challenge_token,
                    challenge: begin.challenge,
                    credential_id: assertion.credentialID,
                    sign_counter: assertion.signCounter,
                    client_data_json: assertion.clientDataJSON,
                    authenticator_data: assertion.authenticatorData,
                    signature: assertion.signature,
                }),
            });
            if (!finishRes.ok) {
                throw new Error(
                    (await finishRes.json()).error || 'Login failed',
                );
            }

            toast.success('Signed in successfully.');
            dispatch('authenticated');
        } catch (error) {
            const message =
                error instanceof Error ? error.message : String(error);
            toast.error(`Sign in failed: ${message}`);
        } finally {
            working = false;
        }
    }

    async function requestEmailLink() {
        if (working || !email.trim()) {
            return;
        }
        working = true;
        try {
            const res = await fetch('/api/auth/email/login/request', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email: email.trim() }),
            });
            const data = await res.json();
            if (!res.ok) {
                throw new Error(data.error || 'Failed to send sign-in link');
            }
            emailSent = true;
        } catch (error) {
            const message =
                error instanceof Error ? error.message : String(error);
            toast.error(message);
        } finally {
            working = false;
        }
    }
</script>

<div class="w-full h-full flex items-center justify-center p-4">
    <Card.Root class="w-full max-w-lg">
        <Card.Header>
            <Card.Title>Sign In</Card.Title>
            <Card.Description>Access your workspace.</Card.Description>
        </Card.Header>
        <Card.Content class="space-y-4">
            {#if !emailMode}
                <Button class="w-full" onclick={signIn} disabled={working}>
                    {working ? 'Signing In...' : 'Continue with Passkey'}
                </Button>
                {#if magicLinkEnabled}
                    <div class="relative">
                        <div class="absolute inset-0 flex items-center">
                            <span class="w-full border-t"></span>
                        </div>
                        <div
                            class="relative flex justify-center text-xs uppercase">
                            <span class="bg-card px-2 text-muted-foreground"
                                >or</span>
                        </div>
                    </div>
                    <Button
                        variant="outline"
                        class="w-full"
                        onclick={() => {
                            emailMode = true;
                            emailSent = false;
                            email = '';
                        }}>
                        Sign in with Email Link
                    </Button>
                {/if}
            {:else if emailSent}
                <p class="text-sm text-center text-muted-foreground">
                    If an account is registered with that address, a sign-in
                    link has been sent. Check your inbox.
                </p>
                <Button
                    variant="outline"
                    class="w-full"
                    onclick={() => {
                        emailSent = false;
                        email = '';
                    }}>
                    Send Another Link
                </Button>
                <Button
                    variant="ghost"
                    class="w-full"
                    onclick={() => {
                        emailMode = false;
                        emailSent = false;
                    }}>
                    Back to Passkey Sign-In
                </Button>
            {:else}
                <Input
                    type="email"
                    placeholder="you@example.com"
                    bind:value={email}
                    disabled={working}
                    onkeydown={(e) => {
                        if (e.key === 'Enter') requestEmailLink();
                    }} />
                <Button
                    class="w-full"
                    onclick={requestEmailLink}
                    disabled={working || !email.trim()}>
                    {working ? 'Sending...' : 'Send Sign-In Link'}
                </Button>
                <Button
                    variant="ghost"
                    class="w-full"
                    onclick={() => {
                        emailMode = false;
                        email = '';
                    }}>
                    Back to Passkey Sign-In
                </Button>
            {/if}
        </Card.Content>
    </Card.Root>
</div>
