import React, { useState, useEffect } from 'react';
import {Run, Kill, EnableRoutes, DisableRoutes} from "@bindings/vpngui/internal/app/xray-core/xraycore.js";
import {Get} from "@bindings/vpngui/internal/app/config/config.js";
import {GetConfig} from "@bindings/vpngui/internal/app/repository/configrepository.js";
import {CaptureTraffic, GetTraffic} from "@bindings/vpngui/internal/app/stats/traffic.js";
import {Countries} from "@constants/countries.jsx";
import {VpnStatuses} from "@constants/vpnStatuses.jsx";
import '@styles/pages/home.css'
import { toast } from 'react-toastify';
import ToggleSwitch from "@components/specific/ToggleSwitch.jsx";
import TrafficMonitor from "@components/specific/TrafficMonitor.jsx";
import RouteCheckbox from "@components/specific/DisableRoutesCheckbox.jsx";
import {formatBytes} from "@utils/formatBytes.js";
import {GetSettings} from "@bindings/vpngui/internal/app/repository/settingsrepository.js";

function PageMain() {
    const [isOn, setIsOn] = useState(false);
    const [status, setStatus] = useState(VpnStatuses["off"]);
    const [ip, setIP] = useState("");
    const [countryCode, setCC] = useState("");
    const [isChecked, setIsChecked] = useState(false);
    const [isTrafficProxyUplink, setIsTrafficProxyUplink] = useState("0.00 Kb/s");
    const [isTrafficProxyDownlink, setIsTrafficProxyDownlink] = useState("0.00 Kb/s");
    const [intervalId, setIntervalId] = useState(0)

    const handleToggle = async () => {
        if (!isOn) {
            setStatus(VpnStatuses["waitOn"]);
            await toggleOn();
            setStatus(VpnStatuses["on"]);
        } else {
            setStatus(VpnStatuses["waitOff"]);
            await toggleOff();
            setStatus(VpnStatuses["off"]);
        }
        setIsOn((prev) => !prev);
    };

    const toggleOn = async () => {
        const error = await Run();
        if (JSON.stringify(error) !== '{}') {
            toast.error(`Ошибка: ${JSON.stringify(error)}`, { theme: 'dark' });
        }
    };

    const toggleOff = async () => {
        const error = await Kill(true);
        if (JSON.stringify(error) !== '{}') {
            toast.error(`Ошибка: ${JSON.stringify(error)}`, { theme: 'dark' });
        }
    };

    useEffect(() => {
        const checkVPNStatus = async () => {
            const config = await GetConfig();
            const xray = await Get();

            if (config["ActiveVPN"]) {
                setIsOn(true);
                setStatus(VpnStatuses["on"]);
            }

            if (config["DisableRoutes"]) {
                setIsChecked(true);
            }

            for (const outbound of xray["outbounds"]) {
                if (outbound["tag"] === "proxy") {
                    setIP(outbound["settings"]["vnext"][0]["address"]);
                    setCC(outbound["settings"]["vnext"][0]["country_code"]);
                }
            }
        };

        void checkVPNStatus();
    }, []);

    useEffect(() => {
        const fetchData = async () => {
            const settings = await GetSettings();

            const fetchTrafficData = async () => {
                await CaptureTraffic();

                const proxyUplink = await GetTraffic("proxy", "uplink");
                const proxyDownlink = await GetTraffic("proxy", "downlink");

                setIsTrafficProxyUplink(formatBytes(proxyUplink));
                setIsTrafficProxyDownlink(formatBytes(proxyDownlink));
            };

            if (isOn) {
                setIntervalId(setInterval(fetchTrafficData, settings["StatsUpdateInterval"] * 1000));
            } else {
                clearInterval(intervalId)
            }

            return () => clearInterval(intervalId);
        };

        void fetchData();
    }, [isOn]);

    const handleCheckboxChange = async () => {
        const error = isChecked ? await DisableRoutes() : await EnableRoutes();
        if (JSON.stringify(error) !== '{}') {
            toast.error(`Ошибка: ${JSON.stringify(error)}`, { theme: 'dark' });
        }
        setIsChecked((prev) => !prev);
    };

    return (
        <>
            <ToggleSwitch
                isOn={isOn}
                status={status}
                onToggle={handleToggle}
                country={Countries[countryCode]}
                ip={ip}
            />
            <TrafficMonitor
                uplink={isTrafficProxyUplink}
                downlink={isTrafficProxyDownlink}
            />
            <RouteCheckbox isChecked={isChecked} onChange={handleCheckboxChange} />
        </>
    );
}

export default PageMain;