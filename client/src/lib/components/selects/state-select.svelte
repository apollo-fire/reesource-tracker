<script lang="ts">
  import { SampleState } from "$lib/components/sample";
  import * as Select from "$lib/components/ui/select";
  let {
    bindValue = $bindable(""),
    disabled = false,
    placeholder = "Select state",
    id = "state-select",
    required = false,
    options = [
      {
        value: SampleState.in_use,
        label: "In Use - Sample is actively being used",
      },
      {
        value: SampleState.available,
        label: "Available - Sample is available for use",
      },
      {
        value: SampleState.archived,
        label: "Archived - Sample is no longer available",
      },
      {
        value: SampleState.broken,
        label: "Broken - Sample does not function as required",
      },
      {
        value: SampleState.unassigned,
        label: "Unassigned - This code is not assigned to a sample",
      },
    ],
    filterMode = false,
    showUnassignedOrArchived = true,
  }: {
    bindValue?: string;
    disabled?: boolean;
    placeholder?: string;
    id?: string;
    required?: boolean;
    options?: { value: string; label: string }[];
    filterMode?: boolean;
    showUnassignedOrArchived?: boolean;
  } = $props();

  if (filterMode) {
    options.unshift({
      value: "",
      label: "Any",
    });
  }
  let filteredOptions = $state(options);
  $effect(() => {
    if (!showUnassignedOrArchived) {
      filteredOptions = options.filter(
        (option) => option.value !== "unassigned" && option.value !== "archived"
      );
    } else {
      filteredOptions = options;
    }
  });
</script>

<Select.Root bind:value={bindValue} type="single" {disabled} {required}>
  <Select.Trigger {id} class="w-full">
    {bindValue
      ? filteredOptions.find((o) => o.value === bindValue)?.label
      : placeholder}
  </Select.Trigger>
  <Select.Content class="w-full">
    {#each filteredOptions as option}
      <Select.Item value={option.value}>{option.label}</Select.Item>
    {/each}
  </Select.Content>
</Select.Root>
