<script lang="ts">
    import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';

    let {
        ref = $bindable(null),
        value = $bindable(),
        ...restProps
    }: DropdownMenuPrimitive.RadioGroupProps = $props();
</script>

<DropdownMenuPrimitive.RadioGroup
    bind:ref={ref}
    bind:value={value}
    data-slot="dropdown-menu-radio-group"
    {...restProps} />
