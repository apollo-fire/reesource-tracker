import { type Writable, get } from 'svelte/store';

import type { AppData } from '$lib/components/app_store';
import { Base64UUIDToString } from '$lib/components/id_helper';
import { SampleState } from '$lib/components/sample';

export class User {
    public id: string;
    public name: string;
    private app_store: Writable<AppData>;
    constructor(data: Record<string, unknown>, app_store: Writable<AppData>) {
        this.id =
            typeof data.ID === 'string' ? Base64UUIDToString(data.ID) : '';
        this.name = typeof data.Name === 'string' ? data.Name : '';
        this.app_store = app_store;
    }

    get AssignedSamples(): Record<string, number> {
        const samples = get(this.app_store).samples || [];
        const userSamples = samples.filter(
            (s) => s.Owner && s.Owner.id === this.id,
        );
        const counts: Record<string, number> = {};
        for (const state of Object.values(SampleState)) {
            counts[state] = userSamples.filter((s) => s.state === state).length;
        }
        return counts;
    }
}
