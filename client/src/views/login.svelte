<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { toast } from 'svelte-sonner';

    import { getPasskeyAssertion } from '$lib/auth/webauthn';
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';

    const dispatch = createEventDispatcher<{ authenticated: void }>();

    let working = $state(false);

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
</script>

<div class="w-full h-full flex items-center justify-center p-4">
    <Card.Root class="w-full max-w-lg">
        <Card.Header>
            <Card.Title>Sign In</Card.Title>
            <Card.Description>Access your workspace.</Card.Description>
        </Card.Header>
        <Card.Content class="space-y-4">
            <Button class="w-full" onclick={signIn} disabled={working}>
                {working ? 'Signing In...' : 'Continue'}
            </Button>
        </Card.Content>
    </Card.Root>
</div>
