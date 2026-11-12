const ContainerWithResizer = {

    create(e) {

        const id = "resizer-" + (++ContainerWithResizer._id);

        const resizer = {...ContainerWithResizer}
        resizer.container = e;

        resizer.container.setAttribute("id", id);

        resizer.sidebar = resizer.container.querySelector('.sidebar')
        if (!resizer.sidebar)
            throw "Sidebar not found: .sidebar"

        const resizeBar = resizer.container.querySelector('.sidebarResizer')
        if (!resizeBar)
            throw "Sidebar not found: .sidebarResizer"
        resizeBar.addEventListener('pointerdown', (e) => {
            resizer.startSidebarResize(e)
        });

        const openButton = resizer.container.querySelector('.btn-open')
        if (openButton) {
            openButton.setAttribute("data-bs-target", "#" + id + " .sidebar")
        }

        const closeButton = resizer.container.querySelector('.btn-close')
        if (closeButton) {
            closeButton.addEventListener('click', () => {
                resizer.hideSidebar()
            });
        }
    },

    _id: 0,
    id: null,

    container: null,
    sidebar: null,

    sidebarMinWidth: 200,
    sidebarMaxWithPercent: 0.6,


    setSidebarWidth(px) {
        const self = this;

        const max = Math.round(self.container.clientWidth * self.sidebarMaxWithPercent);
        const width = Math.min(Math.max(Math.round(px), self.sidebarMinWidth), max);
        self.container.style.setProperty('--sidebar-width', width + 'px');
    },

    startSidebarResize(event) {
        const self = this;

        const startX = event.clientX;
        const startWidth = self.sidebar.getBoundingClientRect().width;

        const move = (e) => self.setSidebarWidth(startWidth + e.clientX - startX);
        const stop = (e) => {
            self.container.removeEventListener('pointermove', move);
            self.container.removeEventListener('pointerup', stop);
            self.setSidebarWidth(startWidth + e.clientX - startX);
        };
        self.container.addEventListener('pointermove', move);
        self.container.addEventListener('pointerup', stop);
        event.preventDefault();
    },

    showSidebar() {
        const self = this;
        const offcanvas = self.sidebar && window.bootstrap && bootstrap.Offcanvas.getInstance(self.sidebar);
        console.log("offcanvas", offcanvas)
        if (offcanvas)
            offcanvas.show();
    },

    hideSidebar() {
        const self = this;
        const offcanvas = self.sidebar && window.bootstrap && bootstrap.Offcanvas.getInstance(self.sidebar);
        if (offcanvas)
            offcanvas.hide();
    }

};

export default ContainerWithResizer