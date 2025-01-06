import React from 'react';

export function RoutesTextarea({ placeholder, value, disabled }) {
    return (
        <textarea placeholder={placeholder} value={value} disabled={disabled} className="routes-textarea"></textarea>
    );
}

export function RoutesActions({placeholder, value, onChange, onAdd, onDelete, isChecked}) {
    return (
        <div className="routes-action">
            <input
                placeholder={placeholder}
                value={value}
                onChange={onChange}
                disabled={isChecked}
            />
            <a className={`routes-add ${isChecked ? 'inactive' : 'active'}`} onClick={onAdd}>+</a>
            <a className={`routes-del ${isChecked ? 'inactive' : 'active'}`} onClick={onDelete}>-</a>
        </div>
    );
}


export function RoutesSection({isChecked, mode, placeholder, value, inputValue, handleInputChange, handleAdd, handleDelete}) {
    const isActive = mode === 'blacklist' ? !isChecked : isChecked;

    return (
        <section className={`routes-${mode} ${isActive ? 'active' : 'inactive'}`}>
            <RoutesTextarea placeholder={placeholder} value={value} disabled={!isActive} />
            <RoutesActions
                placeholder={`Добавить элемент...`}
                value={inputValue}
                onChange={handleInputChange}
                onAdd={handleAdd}
                onDelete={handleDelete}
                isChecked={!isActive}
            />
        </section>
    );
}