import Axios from "axios";

const client = Axios.create({
    withCredentials: true,
    headers: {
        'Content-Type': 'application/json',
        'X-FSM-Request': '1'
    }
});

// Destructive endpoints require a route-bound confirmation header in addition
// to the authenticated session and management role. Callers add it only after
// their confirmation dialog has been accepted.
export const confirmed = routeName => ({headers: {'X-FSM-Confirm': routeName}});

client.interceptors.response.use(res => res, err => {
    if (Axios.isCancel(err) || err.code === 'ERR_CANCELED') {
        return Promise.reject(err);
    } else if(!err.response) {
        window.flash("Service not available", "red");
    } else if(err.response.status === 502) {
        window.flash("Service not available", "red");
    } else if (err.response.status === 401) {
        window.dispatchEvent(new Event('fsm:unauthorized'));
    } else if (err.response.status !== 401) {
        window.flash(err.response.data, "red");
    }
    return Promise.reject(err);
});

export default client;
