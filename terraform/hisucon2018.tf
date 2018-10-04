provider "openstack" {
  user_name = "CHANGEME"
  password  = "CHANGEME"
  tenant_name = "CHANGEME"
  auth_url = "https://identity.tyo2.conoha.io/v2.0"
}

resource "openstack_compute_instance_v2" "web" {
  count = 1
  name = "CHANGEME"
  image_name = "CHANGEME"
  flavor_name = "g-2gb"
  key_pair = "CHANGEME"
  admin_pass = "CHANGEME"
  security_groups = [
    "default",
    "gncs-ipv4-ssh",
    "gncs-ipv4-web",
    "gncs-ipv6-all",
  ]
  metadata {
    "instance_name_tag" = "CHANGEME",
    "properties" = "{\"vnc_keymap\":\"ja\",\"hw_video_model\":\"vga\",\"hw_vif_model\":\"virtio\",\"hw_disk_bus\":\"virtio\",\"cdrom_path\":\"\"}",
    "backup_set" = "0"
  }
}
