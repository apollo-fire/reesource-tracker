import { Base64UUIDToString } from "$lib/components/id_helper";
import type { AppData } from "$lib/components/app_store";
import { get } from "svelte/store";

export class SampleLocation {
  public id: string;
  public name: string;
  public description: string;
  public parentLocationID: string | null;
  private app_store: SvelteStore<AppData> | null = null;

  constructor(
    location_data: { [key: string]: any },
    app_store: SvelteStore<AppData> | null = null,
  ) {
    this.id = Base64UUIDToString(location_data.ID || "");
    this.name = location_data.Name || "";
    this.description = location_data.Description.String || "";
    this.parentLocationID = Base64UUIDToString(
      location_data.ParentLocationID || "",
    );
    this.app_store = app_store;
  }

  get ParentLocation(): SampleLocation | null {
    if (!this.parentLocationID) {
      return null;
    }
    if (!this.app_store) {
      return null;
    }
    const location = get(this.app_store).locations.find(
      (l) => l.id === this.parentLocationID,
    );
    return location ?? null;
  }

  get ChildLocations(): SampleLocation[] {
    if (!this.app_store) {
      return [];
    }
    return get(this.app_store).locations.filter(
      (l) => l.parentLocationID === this.id,
    );
  }

  get CombinedName(): string {
    if (this.ParentLocation) {
      return `${this.ParentLocation.CombinedName} - ${this.name}`;
    }
    return this.name;
  }
}
