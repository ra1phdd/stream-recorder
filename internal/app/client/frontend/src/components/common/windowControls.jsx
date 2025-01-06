import '@styles/components/windowControls.css'
import {useEffect, useState} from "react";
import {IsGOOSWindows} from "@bindings/vpngui/internal/pkg/app/app.js";
import {Hide} from "@bindings/vpngui/internal/pkg/app/app.js";

function WindowControls() {
    const [isWindows, setIsWindows] = useState(false);

    useEffect(async () => {
        const result = await IsGOOSWindows();
        setIsWindows(result);
    }, []);

    const closeWindow = () => {
        Hide()
    };

    const minimizeWindow = () => {
        Hide()
    };

    return (
        <div className="controls">
            <div className="header-controls"></div>
            <div className="main-controls">
                {isWindows &&
                    <div className="window-controls">
                        <button className="control-button" onClick={minimizeWindow}>–</button>
                        <button className="control-button" onClick={closeWindow}>x</button>
                    </div>
                }
            </div>
        </div>
    )
}

export default WindowControls