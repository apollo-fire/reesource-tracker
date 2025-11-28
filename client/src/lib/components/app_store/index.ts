import { writable } from 'svelte/store';
import type { Writable } from 'svelte/store';

import { SampleLocation } from '$lib/components/location';
import { SampleProduct } from '$lib/components/product';
import { Sample } from '$lib/components/sample';
import { User } from '$lib/components/user';

export type AppData = {
    samples: Sample[];
    locations: SampleLocation[];
    products: SampleProduct[];
    users: User[];
    currentPage: string;
};

const app_data_cache = localStorage.getItem('app_data_cache');
let cached_data: AppData = {
    samples: [],
    locations: [],
    products: [],
    users: [],
    currentPage: 'quick_actions',
};

if (app_data_cache) {
    const data = JSON.parse(app_data_cache);
    cached_data = data;
}

export const AppStore: Writable<AppData> = writable(cached_data);

export async function UpdateLocations() {
    const locations_req = await fetch('/api/locations');
    let new_locations: SampleLocation[] = [];
    if (!locations_req.ok) {
        console.error('Failed to fetch locations:', await locations_req.json());
    } else {
        new_locations = (await locations_req.json()) ?? [];
    }
    AppStore.update((data) => ({
        ...data,
        locations: new_locations.map((location) =>
            location instanceof SampleLocation
                ? location
                : new SampleLocation(
                      location as Record<string, unknown>,
                      AppStore,
                  ),
        ),
    }));
}

export async function UpdateProducts() {
    const products_req = await fetch('/api/products');
    let new_products: SampleProduct[] = [];
    if (!products_req.ok) {
        console.error('Failed to fetch products:', await products_req.json());
    } else {
        new_products = (await products_req.json()) ?? [];
    }
    AppStore.update((data) => ({
        ...data,
        products: new_products
            .map((product) =>
                product instanceof SampleProduct
                    ? product
                    : new SampleProduct(
                          product as Record<string, unknown>,
                          AppStore,
                      ),
            )
            .sort(
                (a, b) =>
                    a.name.localeCompare(b.name) || a.id.localeCompare(b.id),
            ),
    }));
}

export async function UpdateUsers() {
    const users_req = await fetch('/api/users');
    let new_users: Record<string, unknown>[] = [];
    if (!users_req.ok) {
        console.error('Failed to fetch users:', await users_req.json());
    } else {
        new_users = (await users_req.json()) ?? [];
    }
    AppStore.update((data) => ({
        ...data,
        users: new_users.map((user) =>
            user instanceof User
                ? user
                : new User(user as Record<string, unknown>, AppStore),
        ),
    }));
}

export async function UpdateSamples() {
    const samples_req = await fetch('/api/samples');
    let new_samples: Sample[] = [];
    if (!samples_req.ok) {
        console.error('Failed to fetch samples:', await samples_req.json());
    } else {
        new_samples = (await samples_req.json()) ?? [];
    }
    AppStore.update((data) => ({
        ...data,
        samples: new_samples.map((sample) =>
            sample instanceof Sample
                ? sample
                : new Sample(sample as Record<string, unknown>, AppStore),
        ),
    }));
}

export async function UpdateAppStore() {
    await Promise.all([
        UpdateLocations(),
        UpdateProducts(),
        UpdateSamples(),
        UpdateUsers(),
    ]);
}

// Connect to sync eventstream API and update store on events
function connectSyncEvents() {
    const evtSource = new EventSource('/api/sync');
    evtSource.addEventListener('samples_updated', () => {
        console.log('Samples updated event received');
        UpdateSamples();
    });
    evtSource.addEventListener('products_updated', () => {
        console.log('Products updated event received');
        UpdateProducts();
    });
    evtSource.addEventListener('locations_updated', () => {
        console.log('Locations updated event received');
        UpdateLocations();
    });
    evtSource.addEventListener('users_updated', () => {
        console.log('Users updated event received');
        UpdateUsers();
    });
    evtSource.onerror = (err) => {
        console.error('Sync eventstream error', err);
        // Optionally, try to reconnect after a delay
    };
}

// Start sync connection immediately
connectSyncEvents();
