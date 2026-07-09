import { useAuth } from '../contexts/AuthContext';

export default function PermissionGate({ permission, children, fallback = null }) {
    const { hasPermission } = useAuth();

    if (!hasPermission(permission)) {
        return fallback;
    }

    return children;
}
