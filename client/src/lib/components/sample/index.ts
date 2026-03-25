import { IDBlobToString, Base64UUIDToString } from "$lib/components/id_helper";
import type { AppData } from "$lib/components/app_store";
import { get } from "svelte/store";
import type { SampleProduct } from "$lib/components/product";
import type { SampleLocation } from "$lib/components/location";
import { SampleMod } from "$lib/components/sample_mod";
import type { User } from "$lib/components/user";

export enum SampleState {
  in_use = "in_use",
  broken = "broken",
  available = "available",
  archived = "archived",
  unassigned = "unassigned",
  unknown = "unknown",
}

export function GetStateName(state: SampleState): string {
  switch (state) {
    case SampleState.in_use:
      return "In Use";
    case SampleState.broken:
      return "Broken";
    case SampleState.available:
      return "Available";
    case SampleState.archived:
      return "Archived";
    case SampleState.unassigned:
      return "Unassigned";
    default:
      return "Unknown";
  }
}

export class Sample {
  public id: string;
  public state: SampleState;
  public locationId: string;
  public productId: string;
  private display_id: string | null = null;
  private app_store: SvelteStore<AppData> | null = null;
  public mods: SampleMod[];
  public productIssue: string = "";
  private ownerId: string | null = null;
  constructor(
    sample_data: { [key: string]: any },
    app_store?: SvelteStore<AppData>,
  ) {
    this.id = sample_data.ID || "";
    this.state = Object.values(SampleState).includes(sample_data.State)
      ? (sample_data.State as SampleState)
      : SampleState.unknown;
    this.locationId = Base64UUIDToString(sample_data.LocationID || "");
    this.productId = Base64UUIDToString(sample_data.ProductID || "");
    this.app_store = app_store ?? null;
    this.ownerId = sample_data.OwnerID
      ? Base64UUIDToString(sample_data.OwnerID)
      : null;
    this.mods =
      sample_data.mods?.map((m: { [key: string]: any }) => new SampleMod(m)) ??
      [];
    this.productIssue = sample_data.ProductIssue?.String || "";
  }

  get DisplayId(): string {
    if (!this.display_id) {
      this.display_id = IDBlobToString(this.id);
    }
    return this.display_id;
  }

  get Owner(): User | null {
    if (!this.app_store || !this.ownerId) {
      return null;
    }
    const owner = get(this.app_store).users.find((u) => u.id === this.ownerId);
    return owner ?? null;
  }

  get Location(): SampleLocation | null {
    if (!this.app_store) {
      return null;
    }
    const location = get(this.app_store).locations.find(
      (l) => l.id === this.locationId,
    );
    return location ?? null;
  }

  get Product(): SampleProduct | null {
    if (!this.app_store) {
      return null;
    }
    const product = get(this.app_store).products.find(
      (p) => p.id === this.productId,
    );
    return product ?? null;
  }

  get Status(): string {
    return GetStateName(this.state);
  }

  get ModSummary(): string {
    if (!this.mods || this.mods.length === 0) {
      return "No mods";
    }
    const activeMods = this.mods.filter((mod: SampleMod) => !mod.time_removed);
    if (activeMods.length === 0) {
      return "No active mods";
    }
    return activeMods.map((mod: SampleMod) => mod.name).join(", ");
  }
}
