<script lang="ts">
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
</script>

<div class="w-full h-screen flex items-center justify-center bg-background p-6">
    <Card.Root class="w-full max-w-sm">
        <Card.Header class="text-center">
            <Card.Title class="text-2xl">Sign in</Card.Title>
            <Card.Description>
                Use your organisation account to continue.
            </Card.Description>
        </Card.Header>
        <Card.Content>
            <Button
                class="w-full"
                onclick={() => (window.location.href = '/api/auth/login')}>
                Sign in with SSO
            </Button>
        </Card.Content>
    </Card.Root>
</div>
