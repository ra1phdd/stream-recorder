import React from 'react';

const TrafficMonitor = ({ uplink, downlink }) => (
    <div className="traffic">
        <div className="traffic-proxy">
            <div className="traffic-proxy_download">
                <p className="traffic-title">Загрузка</p>
                <p className="traffic-speed">▲ {uplink}</p>
            </div>
            <div className="traffic-proxy_upload">
                <p className="traffic-title">Отдача</p>
                <p className="traffic-speed">▼ {downlink}</p>
            </div>
        </div>
    </div>
);

export default TrafficMonitor;