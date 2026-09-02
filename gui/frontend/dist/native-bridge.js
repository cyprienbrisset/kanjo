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

// kanjoProcessFiles valide et affiche une liste [{name, data(base64)}]. Réutilisé par
// l'évènement de glisser-déposer natif et par la récupération des fichiers de lancement.
function kanjoProcessFiles(files) {
  if (!files || !files.length || typeof window.inspectBytes !== 'function') return;
  return Promise.all(files.map(function (f) {
    var bin = atob(f.data);
    var bytes = new Uint8Array(bin.length);
    for (var i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    return window.inspectBytes(f.name, bytes.buffer);
  })).then(function () {
    if (typeof window.show === 'function') window.show('ken');
    if (typeof window.renderDocList === 'function') window.renderDocList();
  });
}

window.addEventListener('load', function () {
  var app = window.go && window.go.main && window.go.main.App;

  // 2) Fichiers ouverts pendant l'exécution (glisser-déposer natif, association ouverte
  //    alors que l'app tourne déjà) : le Go émet 'kanjo:open-files'. L'écouteur est prêt
  //    avant tout dépôt utilisateur, donc pas de course ici.
  if (window.runtime && typeof window.runtime.EventsOn === 'function') {
    window.runtime.EventsOn('kanjo:open-files', kanjoProcessFiles);
  }

  // 3) Fichiers de lancement (double-clic sur une association) : modèle « pull ». On les
  //    réclame une fois l'écouteur enregistré. Émettre depuis Go via OnDomReady posait une
  //    course : l'évènement partait avant l'enregistrement de l'écouteur et les lots
  //    volumineux étaient silencieusement perdus (affichage vide au chargement).
  if (app && typeof app.PendingFiles === 'function') {
    Promise.resolve(app.PendingFiles()).then(kanjoProcessFiles).catch(function () {});
  }
});
