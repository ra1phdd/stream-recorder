import React from 'react';

const RouteCheckbox = ({ isChecked, onChange }) => (
    <div className="disable-routes">
        <label className="checkbox-routes">
            <input type="checkbox" checked={isChecked} onChange={onChange} />
            Отключить маршруты
        </label>
    </div>
);

export default RouteCheckbox;