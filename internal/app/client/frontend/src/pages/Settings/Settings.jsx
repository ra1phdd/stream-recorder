import React, {useState, useEffect, useRef} from 'react';
import '@styles/pages/settings.css';

function PageSettings() {
    return (
        <div className="settings">
            <div className="settings-interface">
                <div className="settings-interface__language">
                    <span>Язык приложения</span>
                </div>
                <div className="settings-interface__theme">
                    <span>Тема приложения</span>
                </div>
            </div>
            <div className="settings-autostart">
                <div className="settings-autostart__startup">
                    <span>Запуск при старте системы</span>
                </div>
                <div className="settings-autostart__hide-on-startup">
                    <span>Скрывать окно при автозапуске</span>
                </div>
            </div>
            <div className="settings-transport">
                <span>Режим работы</span>
            </div>
            <div className="settings-cfg">
                <div className="settings-cfg__logger">
                    <span>Уровень логгирования</span>
                </div>
                <div className="settings-cfg__stats-interval">
                    <span>Время обновления статистики (в секундах)</span>
                </div>
            </div>
        </div>
    );
}

export default PageSettings;