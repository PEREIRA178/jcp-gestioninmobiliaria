(function () {
  'use strict';

  var JCP_BLUE = '#1D4ED8';
  var listingMap = null;
  var pinLayer = null;
  var markersBySlug = {};

  function makePin(color, size) {
    size = size || 28;
    return L.divIcon({
      className: '',
      html:
        '<div style="width:' + size + 'px;height:' + size + 'px;' +
        'border-radius:50% 50% 50% 0;transform:rotate(-45deg);' +
        'background:' + color + ';border:2.5px solid white;' +
        'box-shadow:0 3px 10px rgba(0,0,0,.28)"></div>',
      iconSize: [size, size],
      iconAnchor: [size / 2, size],
      popupAnchor: [0, -size],
    });
  }

  function makeBigPin(color) {
    return L.divIcon({
      className: '',
      html:
        '<div style="width:40px;height:40px;' +
        'border-radius:50% 50% 50% 0;transform:rotate(-45deg);' +
        'background:' + color + ';border:3px solid white;' +
        'box-shadow:0 4px 14px rgba(29,78,216,.4)">' +
        '<div style="transform:rotate(45deg);width:12px;height:12px;' +
        'background:white;border-radius:50%;margin:auto;margin-top:10px"></div>' +
        '</div>',
      iconSize: [40, 40],
      iconAnchor: [20, 40],
      popupAnchor: [0, -44],
    });
  }

  function tileLayer() {
    return L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
      maxZoom: 19,
    });
  }

  // ── Listing map ──────────────────────────────────────────────────
  function initListingMap() {
    var el = document.getElementById('prop-map');
    if (!el || listingMap) return;

    listingMap = L.map('prop-map', { zoomControl: true, scrollWheelZoom: false }).setView([-33.45, -70.65], 10);
    tileLayer().addTo(listingMap);
    pinLayer = L.layerGroup().addTo(listingMap);
    syncListingPins();
  }

  function syncListingPins() {
    if (!listingMap || !pinLayer) return;
    pinLayer.clearLayers();
    markersBySlug = {};

    var cards = document.querySelectorAll('[data-lat][data-lng]');
    var bounds = [];

    cards.forEach(function (card) {
      var lat = parseFloat(card.dataset.lat);
      var lng = parseFloat(card.dataset.lng);
      if (!lat || !lng) return;

      var title = card.dataset.title || '';
      var price = card.dataset.price || '';
      var tipo  = card.dataset.tipo  || '';
      var slug  = card.dataset.slug  || card.dataset.id || '#';
      var commune = card.dataset.commune || '';

      var marker = L.marker([lat, lng], { icon: makePin(JCP_BLUE) });
      markersBySlug[slug] = marker;
      marker.bindPopup(
        '<div class="map-popup">' +
        '<div class="map-popup-type">' + tipo + '</div>' +
        '<div class="map-popup-title">' + title + '</div>' +
        '<div class="map-popup-price">' + price + '</div>' +
        (commune ? '<div class="map-popup-loc"><span class="ms ms-sm">location_on</span>' + commune + '</div>' : '') +
        '<a href="/propiedades/' + slug + '" style="display:block;margin-top:10px;padding:8px 14px;background:#1D4ED8;color:#fff;border-radius:8px;font-size:13px;font-weight:600;text-decoration:none;text-align:center">Ver detalle →</a>' +
        '</div>',
        { maxWidth: 220 }
      );

      marker.on('mouseover', function () { this.openPopup(); });
      marker.on('popupopen', function () {
        document.querySelectorAll('.prop-card').forEach(function (c) { c.classList.remove('map-highlighted'); });
        card.classList.add('map-highlighted');
        card.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      });
      marker.on('popupclose', function () {
        card.classList.remove('map-highlighted');
      });

      pinLayer.addLayer(marker);
      bounds.push([lat, lng]);
    });

    if (bounds.length > 0) {
      listingMap.fitBounds(bounds, { padding: [40, 40], maxZoom: 13 });
    }
  }

  // ── Detail map ───────────────────────────────────────────────────
  function initDetailMap() {
    var el = document.querySelector('[data-map="detail"]');
    if (!el) return;
    var lat = parseFloat(el.dataset.lat);
    var lng = parseFloat(el.dataset.lng);
    if (!lat || !lng) return;

    var map = L.map(el, { scrollWheelZoom: false }).setView([lat, lng], 14);
    tileLayer().addTo(map);
    L.marker([lat, lng], { icon: makeBigPin(JCP_BLUE) })
      .addTo(map)
      .bindPopup(el.dataset.title || 'Propiedad')
      .openPopup();
  }

  function initDetailMiniMap() {
    var el = document.querySelector('[data-map="detail-mini"]');
    if (!el) return;
    var lat = parseFloat(el.dataset.lat);
    var lng = parseFloat(el.dataset.lng);
    if (!lat || !lng) return;

    var map = L.map(el, {
      zoomControl: false,
      attributionControl: false,
      dragging: false,
      scrollWheelZoom: false,
    }).setView([lat, lng], 13);
    tileLayer().addTo(map);
    L.circle([lat, lng], {
      color: JCP_BLUE, fillColor: '#3B82F6', fillOpacity: 0.15, radius: 800, weight: 2,
    }).addTo(map);
    L.marker([lat, lng], { icon: makeBigPin(JCP_BLUE) }).addTo(map);
  }

  // ── Init ─────────────────────────────────────────────────────────
  function init() {
    initListingMap();
    initDetailMap();
    initDetailMiniMap();
  }

  document.addEventListener('DOMContentLoaded', init);

  document.addEventListener('htmx:afterSettle', function () {
    if (listingMap) {
      listingMap.invalidateSize();
      syncListingPins();
    }
  });

  window.addEventListener('resize', function () {
    if (listingMap) listingMap.invalidateSize();
  }, { passive: true });

  document.addEventListener('mouseover', function (e) {
    if (!listingMap) return;
    var card = e.target.closest('.prop-card[data-lat]');
    if (!card) return;
    var slug = card.dataset.slug || card.dataset.id;
    var m = markersBySlug[slug];
    if (m) m.openPopup();
  });

  window.JCPMaps = { syncListingPins: syncListingPins };
}());
