<script lang="ts">
    import { onMount } from 'svelte';

    import { AppStore, UpdateAppStore } from '$lib/components/app_store';
    import UserMenu from '$lib/components/auth/user_menu.svelte';
    import { Base64UUIDToString } from '$lib/components/id_helper';
    import { Button } from '$lib/components/ui/button';
    import { Toaster } from '$lib/components/ui/sonner/index.js';
    import { toast } from 'svelte-sonner';
    import * as Tabs from '$lib/components/ui/tabs';
    import ManageRegistrationsDialog from '$lib/components/user/manage_registrations_dialog.svelte';

    import BootstrapSetup from '$views/bootstrap_setup.svelte';
    import BulkApply from '$views/bulk_apply.svelte';
    import FindSample from '$views/find_sample.svelte';
    import LocationEditor from '$views/location_editor.svelte';
    import LoginView from '$views/login.svelte';
    import PasskeyAssignment from '$views/passkey_assignment.svelte';
    import ProductEditor from '$views/product_editor.svelte';
    import SampleCodeGenerator from '$views/sample_code_generator.svelte';
    import SampleEditor from '$views/sample_editor.svelte';
    import SampleList from '$views/sample_list.svelte';
    import UserEditor from '$views/user_editor.svelte';

    let authReady = $state(false);
    let authenticated = $state(false);
    let bootstrapRequired = $state(false);
    let assignmentToken = $state('');
    let currentUserId = $state('');
    let currentUserName = $state('');
    let currentUserRoles = $state<string[]>([]);
    let manageSelfRegistrationsOpen = $state(false);

    function hasRole(role: string) {
        return currentUserRoles.some((r) => r.toLowerCase() === role);
    }

    let isAdmin = $derived(hasRole('admin'));
    let isMaintainer = $derived(isAdmin || hasRole('maintainer'));

    function canAccessPage(page: string) {
        switch (page) {
            case 'product_edit':
            case 'location_edit':
            case 'sample_code_generator':
                return isMaintainer;
            case 'user_edit':
                return isAdmin;
            default:
                return true;
        }
    }

    async function refreshAuthState() {
        const params = new URLSearchParams(window.location.search);
        assignmentToken = params.get('assignment_token') ?? '';

        const [sessionRes, bootstrapRes] = await Promise.all([
            fetch('/api/auth/session'),
            fetch('/api/auth/bootstrap-status'),
        ]);

        const session = sessionRes.ok
            ? await sessionRes.json()
            : { authenticated: false };
        const bootstrap = bootstrapRes.ok
            ? await bootstrapRes.json()
            : { bootstrap_required: false };

        authenticated = !!session.authenticated;
        currentUserId =
            typeof session?.user?.ID === 'string'
                ? Base64UUIDToString(session.user.ID)
                : '';
        currentUserName = session?.user?.Name ?? '';
        currentUserRoles = Array.isArray(session?.roles) ? session.roles : [];
        bootstrapRequired = !!bootstrap.bootstrap_required;
        if (!assignmentToken && bootstrap.assignment_token) {
            assignmentToken = bootstrap.assignment_token;
        }
    }

    function setAssignmentToken(token: string) {
        assignmentToken = token;
        const url = new URL(window.location.href);
        if (token) {
            url.searchParams.set('assignment_token', token);
        } else {
            url.searchParams.delete('assignment_token');
        }
        window.history.replaceState({}, '', url.toString());
    }

    async function handleBootstrapTokenSelected(
        e: CustomEvent<{ assignmentToken: string }>,
    ) {
        setAssignmentToken(e.detail.assignmentToken);
    }

    async function handlePasskeyAssigned() {
        setAssignmentToken('');
        await handleAuthenticated();
    }

    async function signOut() {
        await fetch('/api/auth/logout', { method: 'POST' });
        authenticated = false;
        currentUserId = '';
        currentUserName = '';
        currentUserRoles = [];
        await refreshAuthState();
    }

    async function handleAuthenticated() {
        await refreshAuthState();
        await UpdateAppStore();
    }

    onMount(async () => {
        // Consume a magic sign-in token before checking session state.
        const initParams = new URLSearchParams(window.location.search);
        const magicToken = initParams.get('magic_token');
        let magicTokenFailed = false;
        if (magicToken) {
            const cleanUrl = new URL(window.location.href);
            cleanUrl.searchParams.delete('magic_token');
            window.history.replaceState({}, '', cleanUrl.toString());

            const res = await fetch('/api/auth/email/login/consume', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ token: magicToken }),
            });
            if (!res.ok) {
                magicTokenFailed = true;
            }
        }

        await refreshAuthState();

        if (window.location.search.includes('sample')) {
            $AppStore.currentPage = 'sample_edit';
        } else if (window.location.search.includes('product')) {
            $AppStore.currentPage = 'product_edit';
        } else {
            if (window.outerWidth * 1.5 < window.outerHeight) {
                $AppStore.currentPage = 'quick_actions';
            } else {
                $AppStore.currentPage = 'sample_list';
            }
        }

        if (authenticated) {
            await UpdateAppStore();
        }

        authReady = true;

        if (magicTokenFailed) {
            toast.error('Sign-in link was invalid or has already been used.');
        }
    });

    $effect(() => {
        if (!authenticated || !authReady) {
            return;
        }

        if (!canAccessPage($AppStore.currentPage)) {
            $AppStore.currentPage = 'sample_list';
        }
    });

    let bulk_apply_active = $derived($AppStore.currentPage === 'bulk_apply');
    let find_sample_active = $derived($AppStore.currentPage === 'find_sample');

    const userDisplayName = $derived(currentUserName || 'Unknown User');
    const userRolesDisplay = $derived(
        currentUserRoles.length > 0 ? currentUserRoles.join(', ') : 'No roles',
    );
</script>

<div class="toaster-wrapper">
    <Toaster position="bottom-center" />
</div>

<main
    class="w-full overflow-hidden p-6 flex flex-col justify-stretch overflow-hidden">
    {#if !authReady}
        <div class="w-full h-full flex items-center justify-center">
            Checking authentication...
        </div>
    {:else if bootstrapRequired && !assignmentToken}
        <BootstrapSetup on:tokenSelected={handleBootstrapTokenSelected} />
    {:else if bootstrapRequired || assignmentToken}
        <PasskeyAssignment
            assignmentToken={assignmentToken}
            on:completed={handlePasskeyAssigned} />
    {:else if !authenticated}
        <LoginView on:authenticated={handleAuthenticated} />
    {:else if $AppStore.currentPage === 'quick_actions'}
        <div class="flex items-center justify-between gap-3 mb-4">
            <div class="text-sm text-muted-foreground">Quick Actions</div>
            <UserMenu
                userDisplayName={userDisplayName}
                userRolesDisplay={userRolesDisplay}
                showManageRegistrations={true}
                on:manageRegistrations={() =>
                    (manageSelfRegistrationsOpen = true)}
                on:signOut={signOut} />
        </div>
        <div
            class="flex flex-col gap-4 items-stretch w-full h-full justify-center p-2">
            <Button
                size="lg"
                class="text-lg py-6"
                onclick={() => ($AppStore.currentPage = 'find_sample')}
                >Find Sample</Button>
            <Button
                size="lg"
                class="text-lg py-6"
                onclick={() => ($AppStore.currentPage = 'bulk_apply')}
                >Bulk Apply</Button>
            <Button
                size="lg"
                class="text-lg py-6"
                onclick={() => ($AppStore.currentPage = 'sample_list')}
                >Sample List</Button>
            {#if isMaintainer}
                <Button
                    size="lg"
                    class="text-lg py-6"
                    onclick={() =>
                        ($AppStore.currentPage = 'sample_code_generator')}
                    >Provision Sample Codes</Button>
                <Button
                    size="lg"
                    class="text-lg py-6"
                    onclick={() => ($AppStore.currentPage = 'product_edit')}
                    >Products</Button>
                <Button
                    size="lg"
                    class="text-lg py-6"
                    onclick={() => ($AppStore.currentPage = 'location_edit')}
                    >Locations</Button>
            {/if}
        </div>
    {:else}
        <Tabs.Root
            bind:value={$AppStore.currentPage}
            class="w-full grow flex flex-col h-full max-h-[100vh]">
            <div class="flex items-center gap-2 max-w-full overflow-x-auto">
                <Tabs.List class="flex-shrink-0">
                    <Tabs.Trigger value="find_sample" class="w-full"
                        >Find Sample</Tabs.Trigger>
                    <Tabs.Trigger value="bulk_apply" class="w-full"
                        >Bulk Apply</Tabs.Trigger>
                    <Tabs.Trigger value="sample_list">Sample List</Tabs.Trigger>
                    {#if isMaintainer}
                        <Tabs.Trigger value="sample_code_generator"
                            >Provision Sample Codes</Tabs.Trigger>
                        <Tabs.Trigger value="product_edit"
                            >Products</Tabs.Trigger>
                        <Tabs.Trigger value="location_edit"
                            >Locations</Tabs.Trigger>
                    {/if}
                    {#if isAdmin}
                        <Tabs.Trigger value="user_edit">Users</Tabs.Trigger>
                    {/if}
                    {#if window.location.search.split('sample_id=').length >= 2}
                        <Tabs.Trigger value="sample_edit"
                            >Sample {window.location.search.split(
                                'sample_id=',
                            )[1]}</Tabs.Trigger>
                    {/if}
                </Tabs.List>
                <div class="ml-auto flex-shrink-0">
                    <UserMenu
                        userDisplayName={userDisplayName}
                        userRolesDisplay={userRolesDisplay}
                        showManageRegistrations={true}
                        on:manageRegistrations={() =>
                            (manageSelfRegistrationsOpen = true)}
                        on:signOut={signOut} />
                </div>
            </div>

            <Tabs.Content
                value="find_sample"
                class="h-full max-h-full overflow-auto">
                <FindSample bind:active={find_sample_active} />
            </Tabs.Content>
            <Tabs.Content
                value="bulk_apply"
                class="h-full max-h-full overflow-auto">
                <BulkApply bind:active={bulk_apply_active} />
            </Tabs.Content>
            <Tabs.Content
                value="sample_list"
                class="h-full max-h-full overflow-auto">
                <SampleList />
            </Tabs.Content>
            {#if isMaintainer}
                <Tabs.Content
                    value="product_edit"
                    class="h-full max-h-full overflow-auto">
                    <ProductEditor />
                </Tabs.Content>
            {/if}
            <Tabs.Content
                value="sample_edit"
                class="h-full max-h-full overflow-auto">
                <SampleEditor />
            </Tabs.Content>
            {#if isMaintainer}
                <Tabs.Content
                    value="location_edit"
                    class="h-full max-h-full overflow-auto">
                    <LocationEditor />
                </Tabs.Content>
                <Tabs.Content
                    value="sample_code_generator"
                    class="h-full max-h-full overflow-auto">
                    <SampleCodeGenerator />
                </Tabs.Content>
            {/if}
            {#if isAdmin}
                <Tabs.Content
                    value="user_edit"
                    class="h-full max-h-full overflow-auto">
                    <UserEditor />
                </Tabs.Content>
            {/if}
        </Tabs.Root>
    {/if}

    {#if authenticated}
        <ManageRegistrationsDialog
            bind:open={manageSelfRegistrationsOpen}
            userId={currentUserId}
            userLabel={userDisplayName}
            useAdminEndpoints={false} />
    {/if}
</main>

<style>
    main {
        @media print {
            max-height: none;
            overflow: visible;
            height: auto;
        }

        @media screen {
            max-height: 100vh;
            overflow: hidden;
            height: 100vh;
        }
    }

    @media print {
        .toaster-wrapper {
            display: none;
        }
    }
</style>
