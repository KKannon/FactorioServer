import React, {useState} from 'react';

const initials = user => {
    const parts = (user?.name || user?.email || '?').trim().split(/\s+/);
    return `${parts[0]?.[0] || '?'}${parts.length > 1 ? parts[parts.length - 1][0] : ''}`.toUpperCase();
};

const Avatar = ({user}) => {
    const [failed, setFailed] = useState(false);
    if (user?.picture && !failed) {
        return <img key={user.picture} src={user.picture} onError={() => setFailed(true)} referrerPolicy="no-referrer" className="identity-avatar" alt=""/>;
    }
    return <span className="identity-avatar identity-avatar-fallback" aria-hidden="true">{initials(user)}</span>;
};

export default Avatar;
