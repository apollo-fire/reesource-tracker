export const ROLE_OPTIONS = ['admin', 'maintainer', 'user'] as const;

export type UserRole = (typeof ROLE_OPTIONS)[number];

export function isUserRole(role: string): role is UserRole {
    return ROLE_OPTIONS.includes(role as UserRole);
}

export function getPrimaryRole(roles: string[]): UserRole {
    if (roles.includes('admin')) {
        return 'admin';
    }
    if (roles.includes('maintainer')) {
        return 'maintainer';
    }
    return 'user';
}

export function canManageRegistrations(
    isAdmin: boolean,
    currentUserId: string,
    targetUserId: string,
): boolean {
    return isAdmin || (!!currentUserId && currentUserId === targetUserId);
}

export function getAssignmentLinkEndpoint(
    isAdmin: boolean,
    userId: string,
): string {
    return isAdmin
        ? `/api/auth/admin/users/${userId}/assignment-link`
        : '/api/auth/self/assignment-link';
}

export function getPasskeysEndpoint(isAdmin: boolean, userId: string): string {
    return isAdmin
        ? `/api/auth/admin/users/${userId}/passkeys`
        : '/api/auth/self/passkeys';
}

export function getPasskeyRevokeEndpoint(
    isAdmin: boolean,
    credentialId: string,
): string {
    return isAdmin
        ? `/api/auth/admin/passkeys/${credentialId}/revoke`
        : `/api/auth/self/passkeys/${credentialId}/revoke`;
}

export function formatRoleLabel(role: string): string {
    if (!role) {
        return '';
    }
    return role.charAt(0).toUpperCase() + role.slice(1);
}
