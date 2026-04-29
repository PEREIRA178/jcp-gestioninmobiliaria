(function(){
  fetch('/admin/settings/logo-url')
    .then(function(r){return r.text();})
    .then(function(url){
      if(!url||!url.trim())return;
      var el=document.querySelector('.sidebar-brand-icon');
      if(!el)return;
      el.textContent='';
      el.style.cssText='background:transparent;padding:0;overflow:hidden';
      var img=document.createElement('img');
      img.src=url.trim();
      img.alt='Logo JCP';
      img.style.cssText='width:40px;height:40px;object-fit:contain;border-radius:10px;display:block';
      img.onerror=function(){el.textContent='JCP';el.style.cssText='';};
      el.appendChild(img);
    })
    .catch(function(){});
})();
