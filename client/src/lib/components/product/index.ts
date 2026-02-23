import { get } from 'svelte/store';

import type { AppData } from '$lib/components/app_store';
import { Base64UUIDToString } from '$lib/components/id_helper';

export class SampleProduct {
    public id: string;
    public name: string;
    public parentProductID: string | null = null;
    private app_store: SvelteStore<AppData> | null = null;
    public partNumber: string = '';
    constructor(
        product_data: Record<string, unknown>,
        app_store: SvelteStore<AppData> | null = null,
    ) {
        this.id =
            typeof product_data.ID === 'string'
                ? Base64UUIDToString(product_data.ID)
                : '';
        this.name =
            typeof product_data.Name === 'string' ? product_data.Name : '';
        this.parentProductID =
            typeof product_data.ParentProductID === 'string'
                ? Base64UUIDToString(product_data.ParentProductID)
                : null;
        this.app_store = app_store;
        this.partNumber =
            typeof product_data.PartNumber === 'object' &&
            product_data.PartNumber !== null &&
            'String' in product_data.PartNumber &&
            typeof (product_data.PartNumber as Record<string, string>)
                .String === 'string'
                ? (product_data.PartNumber as Record<string, string>).String
                : '';
    }

    get ParentProduct(): SampleProduct | null {
        if (!this.parentProductID) {
            return null;
        }
        if (!this.app_store) {
            return null;
        }
        const product = get(this.app_store).products.find(
            (p) => p.id === this.parentProductID,
        );
        return product ?? null;
    }

    get ChildProducts(): SampleProduct[] {
        if (!this.app_store) {
            return [];
        }
        return get(this.app_store).products.filter(
            (p) => p.parentProductID === this.id,
        );
    }

    get CombinedName(): string {
        if (this.ParentProduct) {
            return `${this.ParentProduct.CombinedName} - ${this.name}`;
        }
        return this.name;
    }
}
