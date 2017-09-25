local P = {}

P.name = "PSD �t�@�C���� exo ��"

P.priority = 0

function P.ondragenter(files, state)
  for i, v in ipairs(files) do
    if v.filepath:match("[^.]+$"):lower() == "psd" then
      -- �t�@�C���̊g���q�� psd �̃t�@�C�����������珈���ł������Ȃ̂� true
      return true
    end
  end
  return false
end

function P.ondragover(files, state)
  -- ondragenter �ŏ����ł������Ȃ��̂� ondragover �ł������ł������Ȃ̂Œ��ׂ� true
  return true
end

function P.ondragleave()
end

function P.ondrop(files, state)
  for i, v in ipairs(files) do
    -- �t�@�C���̊g���q�� psd ��������
    if v.filepath:match("[^.]+$"):lower() == "psd" then
      local filepath = v.filepath
      local filename = filepath:match("[^/\\]+$")

      -- �ꏏ�� pfv �t�@�C����͂�ł��Ȃ������ׂ�
      local psddir = filepath:sub(1, #filepath-#filename)
      for i2, v2 in ipairs(files) do
        if v2.filepath:match("[^.]+$"):lower() == "pfv" then
          local pfv = v2.filepath:match("[^/\\]+$")
          local pfvdir = v2.filepath:sub(1, #v2.filepath-#pfv)
          if psddir == pfvdir then
            -- �����t�H���_�[���� pfv �t�@�C�����ꏏ�ɓ�������ł����̂ŘA��
            filepath = filepath .. "|" .. pfv
            -- ���� pfv �t�@�C���̓h���b�v�����t�@�C������͎�菜���Ă���
            table.remove(files, i2)
            break
          end
        end
      end

      -- �t�@�C���𒼐ړǂݍ��ޑ���� exo �t�@�C����g�ݗ��Ă�
      local proj = GCMZDrops.getexeditfileinfo()
      local exo = [[
[exedit]
width=]] .. proj.width .. "\r\n" .. [[
height=]] .. proj.height .. "\r\n" .. [[
rate=]] .. proj.rate .. "\r\n" .. [[
scale=]] .. proj.scale .. "\r\n" .. [[
length=256
audio_rate=]] .. proj.audio_rate .. "\r\n" .. [[
audio_ch=]] .. proj.audio_ch .. "\r\n" .. [[
[0]
start=1
end=256
layer=1
overlay=1
camera=0
[0.0]
_name=�e�L�X�g
�T�C�Y=1
�\�����x=0.0
�������ɌʃI�u�W�F�N�g=0
�ړ����W��ɕ\������=0
�����X�N���[��=0
B=0
I=0
type=0
autoadjust=0
soft=1
monospace=0
align=0
spacing_x=0
spacing_y=0
precision=1
color=ffffff
color2=000000
font=MS UI Gothic
text=]] .. GCMZDrops.encodeexotext(filename) .. "\r\n" .. [[
[0.1]
_name=�A�j���[�V��������
track0=0.00
track1=100.00
track2=0.00
track3=0.00
check0=100
type=0
filter=2
name=Assign@PSDToolKit
param=]] .. "f=" .. GCMZDrops.encodeluastring(filepath) .. ";" .. "\r\n" .. [[
[0.2]
_name=�W���`��
X=0.0
Y=0.0
Z=0.0
�g�嗦=100.00
�����x=0.0
��]=0.00
blend=0
]]

      local filepath = GCMZDrops.createtempfile("psd", ".exo")
      f, err = io.open(filepath, "wb")
      if f == nil then
        error(err)
      end
      f:write(exo)
      f:close()
      debug_print("["..P.name.."] �� " .. v.filepath .. " �� exo �t�@�C���ɍ����ւ��܂����B���̃t�@�C���� orgfilepath �Ŏ擾�ł��܂��B")
      files[i] = {filepath=filepath, orgfilepath=v.filepath}
    end
  end
  -- ���̃C�x���g�n���h���[�ɂ����������������̂ł����͏�� false
  return false
end

return P