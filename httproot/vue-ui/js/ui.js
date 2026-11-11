export default {

    setSidebarWidth(e, px) {

        const max = Math.round(window.innerWidth * 0.6);
        const width = Math.min(Math.max(Math.round(px), 200), max);
        e.style.setProperty('--sidebar-width', width + 'px');
    },

    startSidebarResize(event) {
        const self = this;
        const app = event.target.closest(".app-with-sidebar");
        const sidebar = app.querySelector('.sidebar')

        const startX = event.clientX;
        const startWidth = sidebar.getBoundingClientRect().width;

        const move = (e) => self.setSidebarWidth(app, startWidth + e.clientX - startX);
        const stop = (e) => {
            window.removeEventListener('pointermove', move);
            window.removeEventListener('pointerup', stop);
            self.setSidebarWidth(app, startWidth + e.clientX - startX);
        };
        window.addEventListener('pointermove', move);
        window.addEventListener('pointerup', stop);
        event.preventDefault();
    },

    // Small screens: close the off-canvas menu after navigating
    hideSidebar(event) {
        const app = event.target.closest(".app-with-sidebar");
        const sidebar = app.querySelector('.sidebar')
        const offcanvas = sidebar && window.bootstrap && bootstrap.Offcanvas.getInstance(sidebar);
        if (offcanvas)
            offcanvas.hide();
    }

};