import { stringify as uuidStringify } from 'uuid';

// Converts a 4-byte Uint8Array or base64 string to Go-style sample ID (xx-xx-xx, base36)
export function IDBlobToString(blob: string | Uint8Array): string {
    let bytes: Uint8Array;
    if (typeof blob === 'string') {
        // Assume base64 string
        const bin = atob(blob);
        if (bin.length !== 4) return blob;
        bytes = new Uint8Array(4);
        for (let i = 0; i < 4; i++) bytes[i] = bin.charCodeAt(i);
    } else {
        if (blob.length !== 4) return '';
        bytes = blob;
    }
    // Go: [0]=part0, [1]=part1, [2]=part2, [3]=unused or extra
    const part0 = bytes[0].toString(36).toUpperCase().padStart(2, '0');
    const part1 = bytes[1].toString(36).toUpperCase().padStart(2, '0');
    const part2 = bytes[2].toString(36).toUpperCase().padStart(2, '0');
    return `${part0}-${part1}-${part2}`.toUpperCase();
}

// Converts Go-style sample ID (xx-xx-xx, base36) to a 4-byte base64 string (last byte is 0)
export function IDStringToBlob(sampleId: string): string {
    // Expect format xx-xx-xx
    const parts = sampleId.split('-');
    if (
        parts.length !== 3 ||
        parts[0].length !== 2 ||
        parts[1].length !== 2 ||
        parts[2].length !== 2
    ) {
        return sampleId;
    }
    const b0 = parseInt(parts[0], 36);
    const b1 = parseInt(parts[1], 36);
    const b2 = parseInt(parts[2], 36);
    const b3 = 0; // unused, for compatibility with Go's 4-byte array
    const bytes = new Uint8Array([b0, b1, b2, b3]);
    return btoa(String.fromCharCode(...bytes)).toUpperCase();
}

export function Base64UUIDToString(base_64_id: string): string {
    let uuidStr = '';
    if (base_64_id && base_64_id !== '') {
        const binary = atob(base_64_id);
        const bytes = new Uint8Array(16);
        for (let i = 0; i < bytes.length; i++) {
            const char_code = binary.charCodeAt(i);
            if (char_code < 0 || char_code > 255) {
                return base_64_id;
            }
            bytes[i] = char_code;
        }
        uuidStr = uuidStringify(bytes);
    }
    return uuidStr;
}
