import React from 'react';

const ToggleSwitch = ({ isOn, status, onToggle, country, ip }) => (
    <div className="main-controls">
        <p className={`toggle-status ${isOn ? 'on' : 'off'}`}>{status}</p>
        <div className={`toggle-switch ${isOn ? 'on' : 'off'}`} onClick={onToggle}>
            <div className="toggle-knob">
                <span className="icon">⏻</span>
            </div>
        </div>
        <p className="toggle-country">{isOn && country !== "" ? country : <>&nbsp;</>}</p>
        <p className="toggle-ip">&nbsp;{isOn && ip !== "" ? `IP: ${ip}` : <>&nbsp;</>}</p>
    </div>
);

export default ToggleSwitch;