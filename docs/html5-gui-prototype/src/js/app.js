// This file contains the JavaScript code for the HTML5 GUI prototype application.

document.addEventListener('DOMContentLoaded', function() {
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', function () {
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(sec => sec.classList.remove('active'));
            btn.classList.add('active');
            const tabId = btn.dataset.tab;
            const tabSection = document.getElementById(tabId);
            if (tabSection) {
                tabSection.classList.add('active');
            }
        });
    });
});