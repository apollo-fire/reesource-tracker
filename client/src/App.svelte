<script lang="ts">
    import { onMount } from 'svelte';

    import { AppStore, UpdateAppStore } from '$lib/components/app_store';
    import { Button } from '$lib/components/ui/button';
    import { Toaster } from '$lib/components/ui/sonner/index.js';
    import * as Tabs from '$lib/components/ui/tabs';

    import BulkApply from '$views/bulk_apply.svelte';
    import FindSample from '$views/find_sample.svelte';
    import LocationEditor from '$views/location_editor.svelte';
    import ProductEditor from '$views/product_editor.svelte';
    import SampleCodeGenerator from '$views/sample_code_generator.svelte';
    import SampleEditor from '$views/sample_editor.svelte';
    import SampleList from '$views/sample_list.svelte';
    import UserEditor from '$views/user_editor.svelte';

    onMount(async () => {
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

        UpdateAppStore();
    });

    let bulk_apply_active = $derived($AppStore.currentPage === 'bulk_apply');
    let find_sample_active = $derived($AppStore.currentPage === 'find_sample');
</script>

<div class="toaster-wrapper">
    <Toaster position="bottom-center" />
</div>

<main
    class="w-full overflow-hidden p-6 flex flex-col justify-stretch overflow-hidden">
    {#if $AppStore.currentPage === 'quick_actions'}
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
        </div>
    {:else}
        <Tabs.Root
            bind:value={$AppStore.currentPage}
            class="w-full grow flex flex-col h-full max-h-[100vh]">
            <div class="max-w-full overflow-x-auto">
                <Tabs.List>
                    <Tabs.Trigger value="find_sample" class="w-full"
                        >Find Sample</Tabs.Trigger>
                    <Tabs.Trigger value="bulk_apply" class="w-full"
                        >Bulk Apply</Tabs.Trigger>
                    <Tabs.Trigger value="sample_list">Sample List</Tabs.Trigger>
                    <Tabs.Trigger value="sample_code_generator"
                        >Provision Sample Codes</Tabs.Trigger>
                    <Tabs.Trigger value="product_edit">Products</Tabs.Trigger>
                    <Tabs.Trigger value="user_edit">Users</Tabs.Trigger>
                    <Tabs.Trigger value="location_edit">Locations</Tabs.Trigger>
                    {#if window.location.search.split('sample_id=').length >= 2}
                        <Tabs.Trigger value="sample_edit"
                            >Sample {window.location.search.split(
                                'sample_id=',
                            )[1]}</Tabs.Trigger>
                    {/if}
                </Tabs.List>
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
            <Tabs.Content
                value="product_edit"
                class="h-full max-h-full overflow-auto">
                <ProductEditor />
            </Tabs.Content>
            <Tabs.Content
                value="sample_edit"
                class="h-full max-h-full overflow-auto">
                <SampleEditor />
            </Tabs.Content>
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
            <Tabs.Content
                value="user_edit"
                class="h-full max-h-full overflow-auto">
                <UserEditor />
            </Tabs.Content>
        </Tabs.Root>
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
