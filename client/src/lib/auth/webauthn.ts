function bytesToBase64Url(bytes: Uint8Array): string {
    let binary = '';
    for (const byte of bytes) {
        binary += String.fromCharCode(byte);
    }
    return btoa(binary)
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=+$/g, '');
}

function base64UrlToBytes(value: string): Uint8Array {
    const normalized = value.replace(/-/g, '+').replace(/_/g, '/');
    const padding = '='.repeat((4 - (normalized.length % 4)) % 4);
    const decoded = atob(normalized + padding);
    const bytes = new Uint8Array(decoded.length);
    for (let i = 0; i < decoded.length; i += 1) {
        bytes[i] = decoded.charCodeAt(i);
    }
    return bytes;
}

function hexToBytes(value: string): Uint8Array {
    if (value.length % 2 !== 0) {
        throw new Error('Invalid hex input length.');
    }
    const bytes = new Uint8Array(value.length / 2);
    for (let i = 0; i < value.length; i += 2) {
        bytes[i / 2] = Number.parseInt(value.slice(i, i + 2), 16);
    }
    return bytes;
}

function challengeToBytes(challenge: string): Uint8Array {
    if (/^[0-9a-fA-F]+$/.test(challenge) && challenge.length % 2 === 0) {
        return hexToBytes(challenge);
    }
    return base64UrlToBytes(challenge);
}

function parseSignCounter(authenticatorData: ArrayBuffer): number {
    const bytes = new Uint8Array(authenticatorData);
    if (bytes.length < 37) {
        return 0;
    }
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    return view.getUint32(33, false);
}

function requireWebAuthnSupport() {
    if (typeof PublicKeyCredential === 'undefined' || !navigator.credentials) {
        throw new Error('This browser does not support passkeys.');
    }
}

export type RegistrationPayload = {
    credentialID: string;
    publicKey: string;
    transports: string[];
    clientDataJSON: string;
};

export async function createPasskeyCredential(
    challenge: string,
    userID: string,
    userName: string,
): Promise<RegistrationPayload> {
    requireWebAuthnSupport();

    const publicKey: PublicKeyCredentialCreationOptions = {
        challenge: challengeToBytes(challenge),
        rp: {
            name: 'Reesource Tracker',
            id: window.location.hostname,
        },
        user: {
            id: new TextEncoder().encode(userID),
            name: userName,
            displayName: userName,
        },
        pubKeyCredParams: [
            { type: 'public-key', alg: -7 },
            { type: 'public-key', alg: -257 },
        ],
        timeout: 60000,
        authenticatorSelection: {
            residentKey: 'preferred',
            userVerification: 'preferred',
        },
        attestation: 'none',
    };

    const result = (await navigator.credentials.create({
        publicKey,
    })) as PublicKeyCredential | null;

    if (!result) {
        throw new Error('Passkey setup was cancelled.');
    }

    const response = result.response as AuthenticatorAttestationResponse;
    const publicKeyBuffer =
        response.getPublicKey?.() ?? response.attestationObject;

    return {
        credentialID: bytesToBase64Url(new Uint8Array(result.rawId)),
        publicKey: bytesToBase64Url(new Uint8Array(publicKeyBuffer)),
        transports: response.getTransports?.() ?? ['internal'],
        clientDataJSON: bytesToBase64Url(new Uint8Array(response.clientDataJSON)),
    };
}

export type LoginPayload = {
    credentialID: string;
    signCounter: number;
    clientDataJSON: string;
    authenticatorData: string;
    signature: string;
    userHandle: string | null;
};

export async function getPasskeyAssertion(
    challenge: string,
): Promise<LoginPayload> {
    requireWebAuthnSupport();

    const publicKey: PublicKeyCredentialRequestOptions = {
        challenge: challengeToBytes(challenge),
        rpId: window.location.hostname,
        userVerification: 'preferred',
        timeout: 60000,
    };

    const result = (await navigator.credentials.get({
        publicKey,
    })) as PublicKeyCredential | null;

    if (!result) {
        throw new Error('Passkey sign-in was cancelled.');
    }

    const response = result.response as AuthenticatorAssertionResponse;

    return {
        credentialID: bytesToBase64Url(new Uint8Array(result.rawId)),
        signCounter: parseSignCounter(response.authenticatorData),
        clientDataJSON: bytesToBase64Url(
            new Uint8Array(response.clientDataJSON),
        ),
        authenticatorData: bytesToBase64Url(
            new Uint8Array(response.authenticatorData),
        ),
        signature: bytesToBase64Url(new Uint8Array(response.signature)),
        userHandle: response.userHandle
            ? bytesToBase64Url(new Uint8Array(response.userHandle))
            : null,
    };
}
