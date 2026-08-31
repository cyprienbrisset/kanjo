; associations.nsh — inclus par le template NSIS de Wails.
!macro CustomInstall
  WriteRegStr HKCR ".xml\OpenWithProgids" "Kanjo.Invoice" ""
  WriteRegStr HKCR "Kanjo.Invoice" "" "Facture électronique"
  WriteRegStr HKCR "Kanjo.Invoice\shell\open\command" "" '"$INSTDIR\Kanjo.exe" "%1"'
!macroend
