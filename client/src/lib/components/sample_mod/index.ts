import { Base64UUIDToString } from '$lib/components/id_helper';

export class SampleMod {
    id: string;
    sample_id: string;
    name: string;
    time_added: Date;
    time_removed: Date | null;

    constructor(mod_data: Record<string, unknown>) {
        this.id =
            typeof mod_data.ID === 'string'
                ? Base64UUIDToString(mod_data.ID)
                : '';
        this.sample_id =
            typeof mod_data.SampleID === 'string' ? mod_data.SampleID : '';
        this.name = typeof mod_data.Name === 'string' ? mod_data.Name : '';
        this.time_added =
            typeof mod_data.TimeAdded === 'string' ||
            typeof mod_data.TimeAdded === 'number'
                ? new Date(mod_data.TimeAdded)
                : new Date();
        if (
            typeof mod_data.TimeRemoved === 'object' &&
            mod_data.TimeRemoved !== null &&
            'Valid' in mod_data.TimeRemoved &&
            (mod_data.TimeRemoved as Record<string, string>).Valid &&
            'Time' in mod_data.TimeRemoved &&
            (typeof (mod_data.TimeRemoved as Record<string, string>).Time ===
                'string' ||
                typeof (mod_data.TimeRemoved as Record<string, string>).Time ===
                    'number')
        ) {
            this.time_removed = new Date(
                (mod_data.TimeRemoved as Record<string, string>).Time,
            );
        } else {
            this.time_removed = null;
        }
    }
}
