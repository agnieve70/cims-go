#define AppName "CIMS"
#ifndef AppVersion
  #define AppVersion "dev"
#endif
#ifndef SourceBundle
  #define SourceBundle "..\..\dist\windows\CIMS"
#endif

[Setup]
AppId={{7D4E5E1B-3D6E-4CB4-8C25-77E0F7C6A101}
AppName={#AppName}
AppVersion={#AppVersion}
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
OutputDir=..\..\dist\windows
OutputBaseFilename=CIMS-Setup-{#AppVersion}
Compression=lzma
SolidCompression=yes
WizardStyle=modern

[Files]
Source: "{#SourceBundle}\*"; DestDir: "{app}"; Flags: recursesubdirs ignoreversion

[Icons]
Name: "{autoprograms}\{#AppName}"; Filename: "{app}\start-cims.bat"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\start-cims.bat"

[Run]
Filename: "{app}\start-cims.bat"; Description: "Start CIMS"; Flags: postinstall shellexec skipifsilent
