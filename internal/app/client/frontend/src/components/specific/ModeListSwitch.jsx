import React from 'react';

function ModeSwitch({ isChecked, handleCheckboxChange }) {
    return (
        <div className="mode-list">
            <span className={`black-list ${isChecked ? 'inactive' : 'active'}`}>черный список</span>
            <input
                className="react-switch-checkbox"
                id="react-switch-new"
                type="checkbox"
                checked={isChecked}
                onChange={handleCheckboxChange}
            />
            <label className="react-switch-label" htmlFor="react-switch-new">
                <span className="react-switch-button" />
            </label>
            <span className={`white-list ${isChecked ? 'active' : 'inactive'}`}>белый список</span>
        </div>
    );
}

export default ModeSwitch;