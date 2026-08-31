'use strict';
// Pont natif du client lourd Kanjō. Chargé UNIQUEMENT par l'application Wails
// (injecté avant app.js) ; jamais référencé par index.html, donc Studio-web l'ignore.
// Il n'a d'effet que si les bindings Wails (window.go) sont présents.

// 1) Sélecteur de fichiers natif : app.js appelle window.kanjoOpenFiles() et attend
//    une Promise<[{name, data(base64)}]>. On l'aliase vers le binding Go OpenFiles.
//    (La fonction est définie tout de suite ; window.go n'est requis qu'à l'appel.)
window.kanjoOpenFiles = function () {
  if (!(window.go && window.go.main && window.go.main.App)) return Promise.resolve([]);
  return window.go.main.App.OpenFiles();
};

// 2) Fichiers ouverts par l'OS (double-clic sur une association, glisser-déposer natif) :
//    le Go émet l'événement 'kanjo:open-files' avec [{name, data(base64)}]. On réutilise
//    les fonctions globales déjà définies par app.js (script classique) pour valider+afficher.
window.addEventListener('load', function () {
  if (!(window.runtime && typeof window.runtime.EventsOn === 'function')) return;
  window.runtime.EventsOn('kanjo:open-files', function (files) {
    if (!files || !files.length || typeof window.inspectBytes !== 'function') return;
    Promise.all(files.map(function (f) {
      var bin = atob(f.data);
      var bytes = new Uint8Array(bin.length);
      for (var i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
      return window.inspectBytes(f.name, bytes.buffer);
    })).then(function () {
      if (typeof window.show === 'function') window.show('ken');
      if (typeof window.renderDocList === 'function') window.renderDocList();
    });
  });
});
