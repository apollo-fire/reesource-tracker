import { get } from 'svelte/store';

import type { AppData } from '$lib/components/app_store';
import { Base64UUIDToString, IDBlobToString } from '$lib/components/id_helper';
import type { SampleLocation } from '$lib/components/location';
import type { SampleProduct } from '$lib/components/product';
import { SampleMod } from '$lib/components/sample_mod';
import type { User } from '$lib/components/user';

export enum SampleState {
    in_use = 'in_use',
    broken = 'broken',
    available = 'available',
    archived = 'archived',
    unassigned = 'unassigned',
    unknown = 'unknown',
}

export function GetStateName(state: SampleState): string {
    switch (state) {
        case SampleState.in_use:
            return 'In Use';
        case SampleState.broken:
            return 'Broken';
        case SampleState.available:
            return 'Available';
        case SampleState.archived:
            return 'Archived';
        case SampleState.unassigned:
            return 'Unassigned';
        default:
            return 'Unknown';
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
    public productIssue: string = '';
    private ownerId: string | null = null;
    constructor(
        sample_data: Record<string, unknown>,
        app_store?: SvelteStore<AppData>,
    ) {
        this.id = typeof sample_data.ID === 'string' ? sample_data.ID : '';
        this.state =
            typeof sample_data.State === 'string' &&
            Object.values(SampleState).includes(
                sample_data.State as SampleState,
            )
                ? (sample_data.State as SampleState)
                : SampleState.unknown;
        this.locationId =
            typeof sample_data.LocationID === 'string'
                ? Base64UUIDToString(sample_data.LocationID)
                : '';
        this.productId =
            typeof sample_data.ProductID === 'string'
                ? Base64UUIDToString(sample_data.ProductID)
                : '';
        this.app_store = app_store ?? null;
        this.ownerId =
            typeof sample_data.OwnerID === 'string'
                ? Base64UUIDToString(sample_data.OwnerID)
                : null;
        this.mods = Array.isArray(sample_data.mods)
            ? sample_data.mods.map(
                  (mod) => new SampleMod(mod as Record<string, unknown>),
              )
            : [];
        this.productIssue =
            typeof sample_data.ProductIssue === 'object' &&
            sample_data.ProductIssue !== null &&
            'String' in sample_data.ProductIssue &&
            typeof (sample_data.ProductIssue as Record<string, string>)
                .String === 'string'
                ? (sample_data.ProductIssue as Record<string, string>).String
                : '';
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
        const owner = get(this.app_store).users.find(
            (u) => u.id === this.ownerId,
        );
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
            return 'No mods';
        }
        const activeMods = this.mods.filter(
            (mod: SampleMod) => !mod.time_removed,
        );
        if (activeMods.length === 0) {
            return 'No active mods';
        }
        return activeMods.map((mod: SampleMod) => mod.name).join(', ');
    }
}
