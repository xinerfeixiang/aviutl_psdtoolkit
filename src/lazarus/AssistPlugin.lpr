library AssistPlugin;

{$mode objfpc}{$H+}
{$CODEPAGE UTF-8}

uses
  AviUtl,
  lua,
  AssistMain,
  PSDToolKitAssist,
  SettingDialog,
  Util;

exports
  GetFilterTableList;

begin
  SetMultiByteConversionCodePage(CP_UTF8);
  Randomize();
end.
