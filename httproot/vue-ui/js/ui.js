export default {

    setSidebarWidth(px, save) {
        const max = Math.round(window.innerWidth * 0.6);
        const width = Math.min(Math.max(Math.round(px), 200), max);
        document.documentElement.style.setProperty('--sidebar-width', width + 'px');
        if (save) {
            try {
                localStorage.setItem('sidebarWidth', String(width));
            } catch (e) {
                // storage not available, width is just not remembered
            }
        }
    },

    startSidebarResize(event) {
        const self = this;
        const startX = event.clientX;
        const startWidth = document.getElementById('sidebar').getBoundingClientRect().width;

        const move = (e) => self.setSidebarWidth(startWidth + e.clientX - startX, false);
        const stop = (e) => {
            window.removeEventListener('pointermove', move);
            window.removeEventListener('pointerup', stop);
            self.setSidebarWidth(startWidth + e.clientX - startX, true);
        };
        window.addEventListener('pointermove', move);
        window.addEventListener('pointerup', stop);
        event.preventDefault();
    },

    // Small screens: close the off-canvas menu after navigating
    hideSidebar() {
        const sidebar = document.getElementById('sidebar');
        const offcanvas = sidebar && window.bootstrap && bootstrap.Offcanvas.getInstance(sidebar);
        if (offcanvas)
            offcanvas.hide();
    }

};