import React, {useEffect, useState} from "react";
import {NavLink, Outlet} from "react-router-dom";
import Button from "./Button";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faBars} from "@fortawesome/free-solid-svg-icons";
import {Flash} from "./Flash";
import Avatar from "./Avatar";
import {t} from "../../identity/preferences";

const Layout = ({identity, handleLogout, serverStatus}) => {

    const [isNavCollapsed, setIsNavCollapsed] = useState(true);

    const Status = ({info}) => {

        let text = t('unknown');
        let color = 'gray-light';

        if (info && info.running) {
            text = t('running');
            color = 'green';
        } else if (info && !info.running) {
            text = t('stopped');
            color = 'red';
        }

        return (
            <div className={`bg-${color} accentuated rounded px-2 py-1 text-black`}>{text}</div>
        )
    }

    const Link = ({children, to, last}) => {
        return (
            <NavLink
                onClick={() => setIsNavCollapsed(true)}
                end
                to={to}
                className={({isActive}) => {
                    return [
                        isActive ? "bg-orange" : "",
                        `hover:glow-orange accentuated bg-gray-light hover:bg-orange text-black font-bold py-2 px-4 w-full block${last ? '' : ' mb-1'}`,
                    ].join(" ")
                }}
            >{children}</NavLink>)
    }

    return (
        <>
            {/*Sidebar*/}
            <div className="w-full md:w-88 md:fixed md:top-0 md:left-0 bg-gray-dark md:h-screen overflow-y-auto">
                <div className="py-4 px-2 accentuated">
                    <div className="mx-4 justify-between flex text-center">
                        <span className="text-dirty-white text-xl">Factorio Server Manager</span>
                        <button
                            className="md:hidden cursor-pointer text-white hover:text-dirty-white"
                            onClick={() => setIsNavCollapsed(!isNavCollapsed)}
                        >
                            <FontAwesomeIcon icon={faBars}/>
                        </button>
                    </div>
                </div>
                <div className={isNavCollapsed ? "hidden md:block" : "block"}>
                    <div className="py-4 px-2 accentuated">
                        <h1 className="text-dirty-white text-lg mb-2 mx-4">{t('status')}</h1>
                        <div className="mx-4 mb-4 text-center">
                            <Status info={serverStatus}/>
                        </div>
                    </div>
                    <div className="py-4 px-2 accentuated">
                        <h1 className="text-dirty-white text-lg mb-2 mx-4">{t('management')}</h1>
                        <div className="text-white text-center rounded-sm bg-black shadow-inner mx-4 p-1">
                            <Link to="/">{t('nav.controls')}</Link>
                            <Link to="/saves" last={!identity?.can_manage}>{t('nav.saves')}</Link>
                            {identity?.can_manage && <>
                                <Link to="/mods">{t('nav.mods')}</Link>
                                <Link to="/server-settings">{t('nav.serverSettings')}</Link>
                                <Link to="/game-settings">{t('nav.gameSettings')}</Link>
                                <Link to="/console">{t('nav.console')}</Link>
                                <Link to="/logs" last={true}>{t('nav.logs')}</Link>
                            </>}
                        </div>
                    </div>
                    <div className="py-4 px-2 accentuated">
                        <h1 className="text-dirty-white text-lg mb-2 mx-4">{t('administration')}</h1>
                        <div className="text-white text-center rounded-sm bg-black shadow-inner mx-4 p-1">
                            <Link to="/help" last={true}>{t('nav.help')}</Link>
                        </div>
                    </div>
                    <div className="py-4 px-2 accentuated">
                        <div className="identity-summary mx-4 mb-3">
                            <Avatar user={identity}/>
                            <div className="identity-copy"><strong>{identity?.name || identity?.email}</strong><small>{identity?.role}</small></div>
                        </div>
                        <div className="text-white text-center rounded-sm bg-black shadow-inner mx-4 p-1">
                            <Button type="danger" className="w-full" onClick={handleLogout}>{t('logout')}</Button>
                        </div>
                    </div>
                    <div className="accentuated-t accentuated-x md:block hidden"/>
                </div>
            </div>

            {/*Main*/}
            <div className="md:ml-88 min-h-screen">
                <div className="container md:mx-auto pt-16 md:px-6">
                    <Outlet />
                    <Flash/>
                </div>
            </div>
        </>
    );
}

export default Layout;
