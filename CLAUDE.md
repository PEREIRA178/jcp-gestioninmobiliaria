# CLAUDE.md - JCP Gestión Inmobiliaria

## 📋 Descripción del Proyecto
**JCP Gestión Inmobiliaria** es una aplicación web simple y profesional para la gestión de propiedades.  
Solo el equipo de la empresa puede subir, editar y gestionar las propiedades.  
Los clientes pueden navegar fácilmente las propiedades y contactarse directamente con la empresa (compra o alquiler).

**Objetivo principal:**  
Que sea **extremadamente fácil y rápido** encontrar propiedades y conectarse con la empresa.

## 🏠 Branding y Estilo Visual
- **Nombre**: JCP Gestión Inmobiliaria
- **Logo**: Casa estilizada en azul + tipografía "JCP" en azul fuerte
- **Paleta de colores** (sobria y profesional):
  - **Primary Blue (JCP Blue)**: El azul exacto del logo (color principal)
  - **Neutral Dark**: #111827 / #1F2937
  - **Neutral Light**: #F8FAFC / #F1F5F9
  - **Gray**: #64748B / #94A3B8
- **Estilo general**: Material Design Expressive + toques de Liquid Glass (Apple) → limpio, elegante y moderno
- **Mobile-first**: Todo pensado primero para celular

## 📍 Funcionalidad de Mapas (nuevo requisito)
- Cada propiedad debe tener ubicación: **ciudad** (Copiapó, Santiago, etc.) y **comuna**
- En el **listado de propiedades** y en la **página de detalle** se debe mostrar un mapa interactivo
- Usar pines/marcadores para identificar fácilmente la zona donde está ubicada cada propiedad
- El mapa debe ser ligero y mobile-friendly (recomendado: **Leaflet.js**)

## 🛠️ Stack Tecnológico
- **Backend**: Go + Fiber
- **Frontend**: HTMX + Templ + HTML-first (mínimo JavaScript posible)
- **Mapas**: Leaflet.js (ligero y compatible con HTMX)
- **Estilo**: CSS con el plugin `frontend-design` respetando la paleta JCP

## 🎨 Reglas de Diseño
- Siempre **mobile-first**
- Interfaz simple, limpia y muy intuitiva
- Búsqueda y filtros muy visibles
- Botones de contacto grandes y claros (WhatsApp, teléfono, formulario)
- Animaciones sutiles y elegantes

## 🧩 Reglas Técnicas
- **HTML-first + HTMX** (mínimo JS posible)
- Usar **Templ** para componentes
- Seguir convenciones de `go-web` y `templui`

## 🔐 Reglas de Negocio
- Solo el equipo de JCP puede crear/editar propiedades
- Los visitantes pueden ver propiedades, buscar, filtrar y ver su ubicación en el mapa

## 🧠 Cómo quiero que trabajes
- Usar **Superpowers** para planificar antes de codificar
- Aprovechar `go-web`, `templui` y `frontend-design`
- Priorizar simplicidad y usabilidad
