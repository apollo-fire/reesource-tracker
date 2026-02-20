<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import QRScanner from "$lib/components/qr_scanner/qr_scanner.svelte";
  import ProductSelect from "$lib/components/selects/product-select.svelte";
  import LocationSelect from "$lib/components/selects/location-select.svelte";
  import StateSelect from "$lib/components/selects/state-select.svelte";
  import { Separator } from "$lib/components/ui/separator";
  import { toast } from "svelte-sonner";
  import { Label } from "$lib/components/ui/label";
  import * as Card from "$lib/components/ui/card";
  import * as InputOTP from "$lib/components/ui/input-otp";
  let scannedIds: string[] = $state([]);
  let selectedProduct = $state("");
  let selectedLocation = $state("");
  import { AppStore } from "$lib/components/app_store";
  let sampleState = $state("");
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Checkbox } from "$lib/components/ui/checkbox/index.js";
  import UserSelect from "$lib/components/selects/user-select.svelte";
  import { Input } from "$lib/components/ui/input";
  let { active = $bindable(false) } = $props();
  // QR scan handler
  function handleQRScan(text: string) {
    if (!active) return;
    let sampleId: string | null = null;
    try {
      // Try to parse as URL and extract sample_id param
      const url = new URL(text);

      const param = url.searchParams.get("sample_id");
      if (param && /^[0-9A-Z]{2}-[0-9A-Z]{2}-[0-9A-Z]{2}$/i.test(param)) {
        sampleId = param.toUpperCase();
      }
    } catch (e) {
      // Not a valid URL, fallback to direct code
      if (/^[0-9A-Z]{2}-[0-9A-Z]{2}-[0-9A-Z]{2}$/i.test(text)) {
        sampleId = text.toUpperCase();
      }
    }
    if (sampleId) {
      // Only add if not already present (case-insensitive)
      const exists = scannedIds.some(
        (id) => id.toUpperCase() === sampleId!.toUpperCase()
      );
      if (!exists) {
        toast.success(`Scanned sample ID: ${sampleId}`);
        scannedIds = [...scannedIds, sampleId];
      }
    }
  }

  let selectedVideoInput: string = $state("");

  // Manual sample code entry
  let manualSample: string = $state("");
  let manualSampleError: string = $state("");

  $effect(() => {
    if (manualSample.length === 6) {
      const formatted =
        `${manualSample.slice(0, 2)}-${manualSample.slice(2, 4)}-${manualSample.slice(4, 6)}`.toUpperCase();
      if (/^[0-9A-Z]{2}-[0-9A-Z]{2}-[0-9A-Z]{2}$/.test(formatted)) {
        // Only add if not already present (case-insensitive)
        const exists = scannedIds.some((id) => id.toUpperCase() === formatted);
        if (!exists) {
          scannedIds = [...scannedIds, formatted];
          toast.success(`Added sample ID: ${formatted}`);
          manualSampleError = "";
        } else {
          manualSampleError = "Sample ID already added.";
        }
      } else {
        manualSampleError = "Invalid sample code format.";
      }
      manualSample = "";
    }
  });

  let updateProduct = $state(false);
  let updateLocation = $state(false);
  let updateState = $state(false);
  let updateOwner = $state(false);
  let updateProductIssue = $state(false);
  let selectedOwner = $state("");
  let productIssue = $state("");
  let modNames: string[] = $state([]);
  let modInput = $state("");
  let modError = $state("");
  let modLoading = $state(false);

  async function applyChanges() {
    if (
      !updateProduct &&
      !updateLocation &&
      !updateState &&
      !updateOwner &&
      !updateProductIssue &&
      modNames.length === 0
    ) {
      toast.error(
        "Please enable at least one field to update or add at least one mod."
      );
      return;
    }
    if (scannedIds.length === 0) {
      toast.error("No samples scanned.");
      return;
    }
    let errors = 0;
    let modErrors = 0;
    for (const id of scannedIds) {
      let productId = selectedProduct;
      let locationId = selectedLocation;
      let state = sampleState;
      let ownerId = selectedOwner;
      let issue = productIssue;
      let sampleFetched = false;
      // If a field is not enabled, fetch the current value for this sample
      if (
        !updateProduct ||
        !updateLocation ||
        !updateState ||
        !updateOwner ||
        !updateProductIssue
      ) {
        const existing_sample = $AppStore.samples.find((s) => s.id === id);
        if (existing_sample) {
          sampleFetched = true;
          if (!updateProduct) productId = existing_sample.Product?.id || "";
          if (!updateLocation) locationId = existing_sample.Location?.id || "";
          if (!updateState) state = existing_sample.state || "";
          if (!updateOwner) ownerId = existing_sample.Owner?.id || "";
          if (!updateProductIssue) issue = existing_sample.productIssue || "";
        }
      }
      // If sample not found and state is not configured, fallback to 'available'
      if (!sampleFetched && !updateState) {
        state = "available";
      }
      // Apply main changes
      if (
        updateProduct ||
        updateLocation ||
        updateState ||
        updateOwner ||
        updateProductIssue
      ) {
        const form = new FormData();
        form.append("product_id", productId);
        form.append("location_id", locationId);
        form.append("state", state);
        form.append("owner_id", ownerId);
        form.append("product_issue", issue);
        const res = await fetch(`/api/sample/${id}`, {
          method: "POST",
          body: form,
        });
        if (!res.ok) errors++;
      }
      // Apply all mods
      for (const mod of modNames) {
        if (mod.trim()) {
          const res = await fetch(`/api/sample/${id}/mods`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name: mod }),
          });
          if (!res.ok) modErrors++;
        }
      }
    }
    let msg = [];
    if (
      updateProduct ||
      updateLocation ||
      updateState ||
      updateOwner ||
      updateProductIssue
    ) {
      if (errors === 0) {
        msg.push("Changes applied to all scanned samples.");
      } else {
        msg.push(`Some updates failed (${errors} errors).`);
      }
    }
    if (modNames.length > 0) {
      if (modErrors === 0) {
        msg.push("All mods added to all scanned samples.");
        modNames = [];
      } else {
        msg.push(`Some mods failed to add (${modErrors} errors).`);
      }
    }
    if (msg.length > 0) {
      toast(msg.join("\n"));
    }
  }

  function clearScanned() {
    scannedIds = [];
    toast("Cleared all scanned sample IDs.");
  }

  function removeScannedId(id: string) {
    scannedIds = scannedIds.filter((sampleId) => sampleId !== id);
    toast(`Removed sample ID: ${id}`);
  }
</script>

<div class="flex flex-col gap-8 min-h-full">
  <div class=" flex flex-row flex-wrap gap-6 justify-stretch items-stretch">
    <Card.Root class="flex-grow">
      <Card.Header>
        <Card.Title>Add Sample</Card.Title>
        <Card.Description>
          Scan a QR code or manually enter a sample ID to add it to the list.
        </Card.Description>
      </Card.Header>
      <Card.Content>
        <QRScanner
          containerId="qr-reader-bulk"
          bind:selectedVideoInput
          onQrCodeScan={handleQRScan}
          autoStart={active}
        />
        <div class="self-center mt-12 flex flex-col items-center gap-6">
          <Label for="manual-id-input">Or manually enter the sample ID</Label>
          <InputOTP.Root
            maxlength={6}
            bind:value={manualSample}
            id="manual-id-input"
            disabled={modLoading}
          >
            {#snippet children({ cells })}
              <InputOTP.Group>
                {#each cells.slice(0, 2) as cell}
                  <InputOTP.Slot {cell} />
                {/each}
              </InputOTP.Group>
              <InputOTP.Separator />
              <InputOTP.Group>
                {#each cells.slice(2, 4) as cell}
                  <InputOTP.Slot {cell} />
                {/each}
              </InputOTP.Group>
              <InputOTP.Separator />
              <InputOTP.Group>
                {#each cells.slice(4, 6) as cell}
                  <InputOTP.Slot {cell} />
                {/each}
              </InputOTP.Group>
            {/snippet}
          </InputOTP.Root>
          {#if manualSampleError}
            <span class="text-red-500 text-sm">{manualSampleError}</span>
          {/if}
        </div>
      </Card.Content>
    </Card.Root>

    <Card.Root class="flex-grow">
      <Card.Header>
        <Card.Title>Sample Updates</Card.Title>
        <Card.Description>
          Select fields using the checkboxes, and set values to update for all
          scanned samples. You can also add mods that will be applied with the
          changes.
        </Card.Description>
      </Card.Header>
      <Card.Content>
        <div class="flex flex-col gap-4 mb-4 h-full">
          <div class="flex flex-row gap-2 w-full">
            <Checkbox
              bind:checked={updateProduct}
              id="product-update-enabled"
            />
            <div class="flex-grow">
              <Label
                class="font-bold flex items-center gap-2 mb-2"
                for="product-update-enabled"
                >Update Product
              </Label>

              <ProductSelect
                bind:bindValue={selectedProduct}
                disabled={!updateProduct}
              />
            </div>
          </div>

          <div class="flex flex-row gap-2">
            <Checkbox
              bind:checked={updateLocation}
              id="location-update-enabled"
            />
            <div class="flex-grow">
              <Label
                class="font-bold flex items-center gap-2 mb-2"
                for="location-update-enabled"
                >Update Location
              </Label>
              <LocationSelect
                bind:bindValue={selectedLocation}
                disabled={!updateLocation}
              />
            </div>
          </div>

          <div class="flex flex-row gap-2">
            <Checkbox bind:checked={updateState} id="state-update-enabled" />
            <div class="flex-grow">
              <Label
                class="font-bold flex items-center gap-2 mb-2"
                for="state-update-enabled"
              >
                Update State
              </Label>
              <StateSelect
                bind:bindValue={sampleState}
                disabled={!updateState}
              />
            </div>
          </div>

          <div class="flex flex-row gap-2">
            <Checkbox
              bind:checked={updateProductIssue}
              id="product-issue-update-enabled"
            />
            <div class="flex-grow">
              <Label
                class="font-bold flex items-center gap-2 mb-2"
                for="product-issue-update-enabled"
              >
                Update PCB Issue
              </Label>
              <Input
                type="text"
                bind:value={productIssue}
                placeholder="PCB Issue"
                id="product-issue-input"
                class="border rounded px-2 py-1 w-full"
                disabled={!updateProductIssue}
              />
            </div>
          </div>

          <div class="flex flex-row gap-2">
            <Checkbox bind:checked={updateOwner} id="owner-update-enabled" />
            <div class="flex-grow">
              <Label
                class="font-bold flex items-center gap-2 mb-2"
                for="owner-update-enabled"
              >
                Update Owner
              </Label>
              <UserSelect
                bind:bindValue={selectedOwner}
                placeholder="Select owner"
                id="owner-select"
                disabled={!updateOwner}
              />
            </div>
          </div>

          <div class="flex flex-col gap-2 flex-grow">
            <div class="flex gap-2 items-center">
              <Input
                type="text"
                class="border rounded px-2 py-1 flex-1"
                placeholder="Add mod (optional, applies with changes)"
                bind:value={modInput}
                onkeydown={(e) => {
                  if (e.key === "Enter" && modInput.trim()) {
                    e.preventDefault();
                    if (!modNames.includes(modInput.trim())) {
                      modNames = [...modNames, modInput.trim()];
                    }
                    modInput = "";
                  }
                }}
                disabled={modLoading}
              />
              <Button
                onclick={() => {
                  if (modInput.trim() && !modNames.includes(modInput.trim())) {
                    modNames = [...modNames, modInput.trim()];
                    modInput = "";
                  }
                }}
                disabled={modLoading || !modInput.trim()}>Add</Button
              >
              {#if modError}
                <span class="text-red-500 text-sm">{modError}</span>
              {/if}
            </div>
            {#if modNames.length > 0}
              <div class="flex flex-wrap gap-2 pb-2 flex-grow">
                {#each modNames as mod, i}
                  <Badge variant="outline">
                    {mod}
                    <Button
                      variant="ghost"
                      class="p-0 min-h-0 h-auto text-red-500 px-2 py-1"
                      onclick={() => {
                        modNames = modNames.filter((_, idx) => idx !== i);
                      }}
                      aria-label="Remove mod">&times;</Button
                    >
                  </Badge>
                {/each}
              </div>
            {/if}
          </div>
          <Separator />
          <Button
            onclick={applyChanges}
            disabled={scannedIds.length === 0 || modLoading}
            >Apply Changes</Button
          >
          <Button
            variant="secondary"
            onclick={clearScanned}
            disabled={scannedIds.length === 0 || modLoading}
            >Clear Scanned</Button
          >
        </div>
      </Card.Content>
    </Card.Root>
  </div>
  <Card.Root class="flex-grow">
    <Card.Header>
      <Card.Title>Sample IDs</Card.Title>
      <Card.Description>
        List of sample IDs scanned or entered manually.
      </Card.Description>
    </Card.Header>
    <Card.Content>
      {#if scannedIds.length === 0}
        <div class="text-muted-foreground">No samples scanned yet.</div>
      {:else}
        <div class="flex flex-row flex-wrap gap-2">
          {#each scannedIds as id}
            <Badge variant="outline" class="mb-2"
              >{id}
              <Button
                variant="ghost"
                class="p-0 min-h-0 h-auto text-red-500 px-2 py-1"
                onclick={() => removeScannedId(id)}
                aria-label="Remove mod">&times;</Button
              ></Badge
            >
          {/each}
        </div>
      {/if}
    </Card.Content>
  </Card.Root>
</div>

<style>
</style>
