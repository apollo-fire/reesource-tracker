import { get } from 'svelte/store';

import type { AppData } from '$lib/components/app_store';
import { Base64UUIDToString } from '$lib/components/id_helper';

export class SampleLocation {
    public id: string;
    public name: string;
    public description: string;
    public parentLocationID: string | null;
    private app_store: SvelteStore<AppData> | null = null;

    constructor(
        location_data: Record<string, unknown>,
        app_store: SvelteStore<AppData> | null = null,
    ) {
        this.id =
            typeof location_data.ID === 'string'
                ? Base64UUIDToString(location_data.ID)
                : '';
        this.name =
            typeof location_data.Name === 'string' ? location_data.Name : '';
        this.description =
            typeof location_data.Description === 'object' &&
            location_data.Description !== null &&
            'String' in location_data.Description &&
            typeof (location_data.Description as Record<string, string>)
                .String === 'string'
                ? (location_data.Description as Record<string, string>).String
                : '';
        this.parentLocationID =
            typeof location_data.ParentLocationID === 'string'
                ? Base64UUIDToString(location_data.ParentLocationID)
                : '';
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
