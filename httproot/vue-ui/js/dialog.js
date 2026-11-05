import {createApp} from 'vue'

export default {

    template: `
            <div class="modal fade" aria-hidden="true">
                <div class="modal-dialog">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title"></h5>
                            <button type="button" class="btn-close dismiss" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body">
                            
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary dismiss footer-dismiss" data-bs-dismiss="modal">Cancel</button>
                            <button type="button" class="btn btn-primary confirm" data-bs-dismiss="modal">OK</button>
                        </div>
                    </div>
                </div>
             </div>`,

    methods: {
        setTitle(title) {
            const titleElement = this.element.querySelector(".modal-title");
            titleElement.innerText = title;
        },
        setDismissText(text) {
            const buttonElement = this.element.querySelector(".footer-dismiss");
            if(text)
                buttonElement.innerText = text;
            else
                buttonElement.style.display = 'none';
        },
        setConfirmText(text) {
            const buttonElement = this.element.querySelector(".confirm");
            buttonElement.innerText = text;
        },
        setStyle(style) {
            const dialogElement = this.element.querySelector(".modal-dialog");
            dialogElement.setAttribute("style", style);
        },
        setContent(html) {
            const bodyElement = this.element.querySelector(".modal-body");
            bodyElement.innerHTML = html;
        },
        setContentComponent(component, createAppArgs) {
            const bodyElement = this.element.querySelector(".modal-body");
            this.app = createApp(component, createAppArgs);
            return this.app.mount(bodyElement);
        },
        addOnDismissListener(f) {
            this.onDismissListener.push(f);
        },
        onDismissClick() {
            for (const f of this.onDismissListener)
                f();
        },
        addOnConfirmListener(f) {
            this.onConfirmListener.push(f);
        },
        onConfirmClick() {
            for (const f of this.onConfirmListener)
                f();
        },
        addOnClosedListener(f) {
            this.onClosedListeners.push(f);
        },
        addOnCloseListener(f) {
            this.onCloseListener.push(f);
        },
        close() {
            this.modal.hide();
        },
        onClose() {
            for (const f of this.onCloseListener)
                f();

            console.log("Close " + this.id);
            if (this.app) {
                console.log("Unmount: " + this.app);
                this.app.unmount();
            }

            for (const f of this.onClosedListeners)
                f();
        }
    },

    _dialogId: 0,

    /**
     *
     */
    create() {

        const dlg = {
            id: 'dialog' + (++this._dialogId),
            onDismissListener: [], //dismiss button click
            onConfirmListener: [], //confirm button click
            onCloseListener: [], //before close
            onClosedListeners: [] //after close
        }

        dlg.element = document.createElement('div');
        dlg.element.setAttribute('id', dlg.id);
        dlg.element.innerHTML = this.template;

        //Assign methods directly to dialog object.
        for (const [key, value] of Object.entries(this.methods)) {
            dlg[key] = value;
        }

        const body = document.getElementsByTagName('body');
        body[0].appendChild(dlg.element);

        dlg.modalElement = dlg.element.querySelector(".modal");
        dlg.modal = new bootstrap.Modal(dlg.modalElement, {backdrop: true});

        dlg.modalElement.addEventListener('hidden.bs.modal', function () {
            dlg.onClose();
            const body = document.getElementsByTagName('body');
            body[0].removeChild(dlg.element);
        });

        const confirmButton = dlg.element.querySelector(".confirm");
        confirmButton.addEventListener('click', () => dlg.onConfirmClick());

        const dismissButtons = dlg.element.querySelectorAll(".dismiss");
        for (const dismissButton of dismissButtons)
            dismissButton.addEventListener('click', () => dlg.onDismissClick());

        dlg.modal.show();

        return dlg;
    },
};