import {NavLink} from "react-router-dom";

function Header() {
    const handleLinkClick = (url) => {
        window.open(url, '_blank');
    };

    return (
        <header>
            <nav>
                <NavLink to="/" className={({isActive}) => (isActive ? 'active' : '')}>
                    Главная
                </NavLink>
                <NavLink to="/routes" className={({isActive}) => (isActive ? 'active' : '')}>
                    Маршруты
                </NavLink>
                <NavLink to="/log" className={({isActive}) => (isActive ? 'active' : '')}>
                    Лог
                </NavLink>
                <NavLink to="/accounts" className={({isActive}) => (isActive ? 'active' : '')}>
                    Аккаунт
                </NavLink>
                <NavLink to="/settings" className={({isActive}) => (isActive ? 'active' : '')}>
                    Настройки
                </NavLink>
                <NavLink to="/faq" className={({isActive}) => (isActive ? 'active' : '')}>
                    FAQ
                </NavLink>
                <a href="#" onClick={() => handleLinkClick('https://t.me/nsvpnsupport_bot')}>Техподдержка</a>
                <a href="#" onClick={() => handleLinkClick('https://t.me/nsvpn_bot')}>Telegram-бот</a>
            </nav>
        </header>
    )
}

export default Header