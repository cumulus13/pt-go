; NSIS installer for pt (Windows, amd64)
; Installs to C:\TOOLS\pt and adds that directory to the SYSTEM PATH.
;
; Build (from repo root, see packaging/build-windows-installer.sh):
;   makensis -DVERSION=1.0.81 -DBIN=pt.exe -DREPO=. -DOUTFILE=dist\pt-setup.exe packaging\windows\pt.nsi

Unicode true
ManifestDPIAware true
SetCompressor /SOLID lzma

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
!ifndef BIN
  !error "Pass -DBIN=<path to pt.exe>"
!endif
!ifndef REPO
  !define REPO "."
!endif
!ifndef OUTFILE
  !define OUTFILE "pt-${VERSION}-windows-amd64-setup.exe"
!endif

!define APPNAME     "pt"
!define PUBLISHER   "Hadi Cahyadi"
!define URL         "https://github.com/cumulus13/pt-go"
!define DEFAULT_DIR "C:\TOOLS\pt"
!define UNINST_KEY  "Software\Microsoft\Windows\CurrentVersion\Uninstall\pt-go"

!include "MUI2.nsh"
!include "x64.nsh"
!include "WinMessages.nsh"
!include "LogicLib.nsh"

Name "${APPNAME} ${VERSION}"
OutFile "${OUTFILE}"
InstallDir "${DEFAULT_DIR}"
RequestExecutionLevel admin
BrandingText "${APPNAME} ${VERSION}"

!ifdef VI_VERSION
  VIProductVersion "${VI_VERSION}"
  VIAddVersionKey "ProductName"     "${APPNAME}"
  VIAddVersionKey "CompanyName"     "${PUBLISHER}"
  VIAddVersionKey "FileDescription" "pt installer"
  VIAddVersionKey "FileVersion"     "${VERSION}"
  VIAddVersionKey "ProductVersion"  "${VERSION}"
  VIAddVersionKey "LegalCopyright"  "MIT License"
!endif

!define MUI_ABORTWARNING
!define MUI_ICON   "${REPO}\pt.ico"
!define MUI_UNICON "${REPO}\pt.ico"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Function .onInit
  ${IfNot} ${RunningX64}
    MessageBox MB_ICONSTOP "This installer requires 64-bit Windows." /SD IDOK
    Abort
  ${EndIf}
  SetRegView 64
FunctionEnd

Function un.onInit
  SetRegView 64
FunctionEnd

; Runs path.ps1 ($0 = add|remove). Result code is checked; failure is reported
; but is not fatal for the file install.
!macro RunPathScript ACTION
  DetailPrint "Updating system PATH (${ACTION})..."
  nsExec::ExecToLog '"$WINDIR\System32\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "$INSTDIR\path.ps1" -Action ${ACTION} -Dir "$INSTDIR"'
  Pop $0
  ${If} $0 != "0"
    DetailPrint "PATH update returned: $0"
    MessageBox MB_ICONEXCLAMATION|MB_OK "Could not update the system PATH automatically (code $0).$\r$\nAdd $INSTDIR to PATH manually if needed." /SD IDOK
  ${EndIf}
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
!macroend

Section "Install"
  SetRegView 64
  SetOutPath "$INSTDIR"
  SetOverwrite on

  File /oname=pt.exe "${BIN}"
  File "${REPO}\pt.yml.example"
  File "${REPO}\README.md"
  File "${REPO}\pt.ico"
  File "${REPO}\VERSION"
  File "${REPO}\path.ps1"

  WriteUninstaller "$INSTDIR\uninstall.exe"

  !insertmacro RunPathScript add

  ; Add/Remove Programs entry
  WriteRegStr   HKLM "${UNINST_KEY}" "DisplayName"          "${APPNAME} (clipboard to file tool)"
  WriteRegStr   HKLM "${UNINST_KEY}" "DisplayVersion"       "${VERSION}"
  WriteRegStr   HKLM "${UNINST_KEY}" "Publisher"            "${PUBLISHER}"
  WriteRegStr   HKLM "${UNINST_KEY}" "URLInfoAbout"         "${URL}"
  WriteRegStr   HKLM "${UNINST_KEY}" "InstallLocation"      "$INSTDIR"
  WriteRegStr   HKLM "${UNINST_KEY}" "DisplayIcon"          "$INSTDIR\pt.exe"
  WriteRegStr   HKLM "${UNINST_KEY}" "UninstallString"      '"$INSTDIR\uninstall.exe"'
  WriteRegStr   HKLM "${UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1
SectionEnd

Section "Uninstall"
  SetRegView 64

  !insertmacro RunPathScript remove

  Delete /REBOOTOK "$INSTDIR\pt.exe"
  Delete /REBOOTOK "$INSTDIR\pt.yml.example"
  Delete /REBOOTOK "$INSTDIR\README.md"
  Delete /REBOOTOK "$INSTDIR\pt.ico"
  Delete /REBOOTOK "$INSTDIR\VERSION"
  Delete /REBOOTOK "$INSTDIR\path.ps1"
  Delete /REBOOTOK "$INSTDIR\uninstall.exe"

  ; Only removes the directory if it is empty (user config/backups are left alone)
  RMDir "$INSTDIR"

  DeleteRegKey HKLM "${UNINST_KEY}"
SectionEnd
